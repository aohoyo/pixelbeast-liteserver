package ssl

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
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

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	legoLog "github.com/go-acme/lego/v4/log"
	"github.com/go-acme/lego/v4/providers/dns/alidns"
	"github.com/go-acme/lego/v4/providers/dns/tencentcloud"
	"github.com/go-acme/lego/v4/registration"
)

// ACME 目录 URL
const (
	CADirLetsEncrypt = lego.LEDirectoryProduction
	CADirLiteSSL     = "https://acme.litessl.com/acme/v2/directory"
)

// PendingChallenge 待验证的挑战状态
type PendingChallenge struct {
	Domain          string
	Email           string
	Provider        string
	ChallengeMethod string
	Client          *lego.Client
	Resource        *certificate.Resource
	ExpiresAt       time.Time
	DNSProvider     challenge.Provider // DNS 验证 provider（API 模式）
}

// DNSChallengeInfo DNS 验证信息（返回给前端）
type DNSChallengeInfo struct {
	FQDN       string `json:"fqdn"`        // _acme-challenge.example.com
	Value      string `json:"value"`       // TXT 记录值
	RecordType string `json:"record_type"` // TXT
}

// FileChallengeInfo 文件验证信息（返回给前端）
type FileChallengeInfo struct {
	Token     string `json:"token"`      // 验证 token
	KeyAuth   string `json:"key_auth"`   // key authorization
	URLPath   string `json:"url_path"`   // /.well-known/acme-challenge/{token}
	VerifyURL string `json:"verify_url"` // http://domain/.well-known/acme-challenge/{token}
}

// CertLogEntry 证书申请日志条目
type CertLogEntry struct {
	Time    string `json:"time"`    // "14:30:05"
	Message string `json:"message"` // "正在注册 ACME 账户..."
	Level   string `json:"level"`   // "info" | "success" | "error" | "warn"
}

// CertProgress 证书申请进度
type CertProgress struct {
	Domain   string         `json:"domain"`
	Step     int            `json:"step"`      // 1-5
	StepText string         `json:"step_text"` // "注册账户"
	Status   string         `json:"status"`    // "running" | "success" | "error" | "waiting"
	Logs     []CertLogEntry `json:"logs"`
}

// ========== SSLManager 进度追踪方法 ==========

// initCertProgress 初始化证书申请进度
func (m *SSLManager) initCertProgress(domain string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.certProgress == nil {
		m.certProgress = make(map[string]*CertProgress)
	}
	m.certProgress[domain] = &CertProgress{
		Domain:   domain,
		Step:     1,
		StepText: "初始化",
		Status:   "running",
		Logs:     []CertLogEntry{},
	}
}

// addCertLog 添加证书申请日志
func (m *SSLManager) addCertLog(domain, level, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.certProgress == nil {
		m.certProgress = make(map[string]*CertProgress)
	}
	p, ok := m.certProgress[domain]
	if !ok {
		p = &CertProgress{Domain: domain, Status: "running", Logs: []CertLogEntry{}}
		m.certProgress[domain] = p
	}
	p.Logs = append(p.Logs, CertLogEntry{
		Time:    time.Now().Format("15:04:05"),
		Message: message,
		Level:   level,
	})
}

// setCertStep 设置证书申请步骤
func (m *SSLManager) setCertStep(domain string, step int, stepText, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.certProgress == nil {
		return
	}
	if p, ok := m.certProgress[domain]; ok {
		p.Step = step
		p.StepText = stepText
		p.Status = status
	}
}

// GetCertProgress 获取证书申请进度
func (m *SSLManager) GetCertProgress(domain string) *CertProgress {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.certProgress == nil {
		return nil
	}
	p, ok := m.certProgress[domain]
	if !ok {
		return nil
	}
	// 返回副本
	cp := *p
	cp.Logs = make([]CertLogEntry, len(p.Logs))
	copy(cp.Logs, p.Logs)
	return &cp
}

// ClearCertProgress 清除证书申请进度
func (m *SSLManager) ClearCertProgress(domain string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.certProgress != nil {
		delete(m.certProgress, domain)
	}
}

// LegoUser 实现 lego registration.User 接口
type LegoUser struct {
	Email string
	Reg   *registration.Resource
	key   crypto.PrivateKey
}

func (u *LegoUser) GetEmail() string                        { return u.Email }
func (u *LegoUser) GetRegistration() *registration.Resource { return u.Reg }
func (u *LegoUser) GetPrivateKey() crypto.PrivateKey        { return u.key }

// getCADirURL 获取 ACME 目录 URL
func getCADirURL(provider string) string {
	switch provider {
	case "litessl":
		return CADirLiteSSL
	default:
		return CADirLetsEncrypt
	}
}

// getOrLoadAccount 加载或创建 ACME 账户
func getOrLoadAccount(provider, email, acmeDir string) (*LegoUser, error) {
	accountDir := filepath.Join(acmeDir, provider)
	if err := os.MkdirAll(accountDir, 0700); err != nil {
		return nil, fmt.Errorf("create account dir: %w", err)
	}

	keyPath := filepath.Join(accountDir, "account.key")
	user := &LegoUser{Email: email}

	// 尝试加载已有私钥
	if keyData, err := os.ReadFile(keyPath); err == nil {
		block, _ := pem.Decode(keyData)
		if block != nil {
			key, err := x509.ParseECPrivateKey(block.Bytes)
			if err == nil {
				user.key = key
				return user, nil
			}
		}
	}

	// 生成新私钥
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate account key: %w", err)
	}
	user.key = key

	// 保存私钥
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("marshal account key: %w", err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	}), 0600); err != nil {
		return nil, fmt.Errorf("save account key: %w", err)
	}

	return user, nil
}

// newLegoClient 创建 lego Client（注册或加载账户）
func newLegoClient(user *LegoUser, provider string, eabKid, eabHmac string) (*lego.Client, error) {
	cfg := lego.NewConfig(user)
	cfg.CADirURL = getCADirURL(provider)

	client, err := lego.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create lego client: %w", err)
	}

	// 注册账户（如果尚未注册）
	if user.Reg == nil {
		if eabKid != "" && eabHmac != "" {
			// 使用 EAB 注册（如 LiteSSL）
			reg, err := client.Registration.RegisterWithExternalAccountBinding(registration.RegisterEABOptions{
				TermsOfServiceAgreed: true,
				Kid:                  eabKid,
				HmacEncoded:          eabHmac,
			})
			if err != nil {
				return nil, fmt.Errorf("register account with EAB: %w", err)
			}
			user.Reg = reg
		} else {
			reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
			if err != nil {
				// 如果已注册，尝试查询
				reg, err2 := client.Registration.QueryRegistration()
				if err2 != nil {
					return nil, fmt.Errorf("register account: %w (query: %w)", err, err2)
				}
				user.Reg = reg
			} else {
				user.Reg = reg
			}
		}
	}

	return client, nil
}

// manualDNSProvider 手动 DNS 验证 provider（不做任何操作，由用户手动添加 TXT 记录）
type manualDNSProvider struct {
	mu      sync.RWMutex
	records map[string]string // fqdn -> value
}

func newManualDNSProvider() *manualDNSProvider {
	return &manualDNSProvider{records: make(map[string]string)}
}

func (p *manualDNSProvider) Present(domain, token, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	p.mu.Lock()
	p.records[info.FQDN] = info.Value
	p.mu.Unlock()
	return nil
}

func (p *manualDNSProvider) CleanUp(domain, token, keyAuth string) error {
	return nil
}

// getDNSChallengeInfo 获取 DNS 挑战信息
func getDNSChallengeInfo(domain, keyAuth string) *DNSChallengeInfo {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	return &DNSChallengeInfo{
		FQDN:       info.FQDN,
		Value:      info.Value,
		RecordType: "TXT",
	}
}

// ========== SSLManager lego 方法 ==========

// ObtainCertificateLego 使用 lego 申请证书（HTTP-01 自动模式）
func (m *SSLManager) ObtainCertificateLego(domain, email, provider string) error {
	m.initCertProgress(domain)
	m.addCertLog(domain, "info", fmt.Sprintf("开始申请证书: %s (提供商: %s)", domain, provider))
	m.setCertStep(domain, 1, "注册账户", "running")

	m.mu.Lock()
	acmeDir := m.acmeDir
	certDir := m.certDir
	m.mu.Unlock()

	m.addCertLog(domain, "info", "正在注册 ACME 账户...")
	user, err := getOrLoadAccount(provider, email, acmeDir)
	if err != nil {
		m.addCertLog(domain, "error", fmt.Sprintf("账户注册失败: %v", err))
		m.setCertStep(domain, 1, "注册账户", "error")
		return err
	}
	m.addCertLog(domain, "success", "ACME 账户就绪")

	eabKid, eabHmac := m.GetEABCredentials()
	client, err := newLegoClient(user, provider, eabKid, eabHmac)
	if err != nil {
		m.addCertLog(domain, "error", fmt.Sprintf("创建 ACME 客户端失败: %v", err))
		m.setCertStep(domain, 1, "注册账户", "error")
		return err
	}

	m.setCertStep(domain, 2, "准备验证", "running")
	m.addCertLog(domain, "info", "正在准备 HTTP-01 验证...")

	// HTTP-01: 使用 selfHostProvider，通过管理面板的端口 80 服务托管验证文件
	// selfHostProvider 将 token/keyAuth 存入 acmeFileTokens，由 GetHTTPSRedirectHandler 响应
	selfProvider := newSelfHostProvider(m)
	if err := client.Challenge.SetHTTP01Provider(selfProvider); err != nil {
		m.addCertLog(domain, "error", fmt.Sprintf("设置 HTTP-01 验证失败: %v", err))
		m.setCertStep(domain, 2, "准备验证", "error")
		return fmt.Errorf("set http01 provider: %w", err)
	}

	m.addCertLog(domain, "success", "HTTP-01 验证已就绪，开始申请...")

	// 申请证书（带超时保护）
	req := certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	}

	m.setCertStep(domain, 4, "验证获取", "running")
	m.addCertLog(domain, "info", "正在向 CA 提交验证请求...")

	oldLogger := m.setLegoProgressLogger(domain)
	certResource, err := obtainWithTimeout(client, req, m, domain)
	restoreLegoLogger(oldLogger)
	if err != nil {
		return fmt.Errorf("obtain certificate: %w", err)
	}

	m.addCertLog(domain, "success", "证书获取成功，正在保存...")

	// 保存证书到磁盘
	if err := m.saveLegoCertResource(domain, certResource, certDir); err != nil {
		return err
	}

	// 加载到内存
	certPath := filepath.Join(certDir, domain+".crt")
	keyPath := filepath.Join(certDir, domain+".key")
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return fmt.Errorf("load obtained cert: %w", err)
	}

	m.mu.Lock()
	m.certs[domain] = &cert
	if _, ok := m.configs[domain]; !ok {
		m.configs[domain] = &config.SSLConfig{Enabled: true}
	}
	cfg := m.configs[domain]
	cfg.Enabled = true
	cfg.AutoHTTPS = true
	cfg.Provider = provider
	cfg.ChallengeMethod = "http-auto"
	cfg.Email = email
	m.updateCertInfoFromCert(domain, cfg, &cert)
	m.mu.Unlock()

	log.Printf("[SSL] lego 证书申请成功: %s (provider: %s)", domain, provider)
	m.setCertStep(domain, 5, "完成", "success")
	m.addCertLog(domain, "success", fmt.Sprintf("证书申请完成！提供商: %s", provider))
	return nil
}

// PrepareFileChallenge 文件验证第一阶段：生成验证文件信息
func (m *SSLManager) PrepareFileChallenge(domain, email, provider string) (*FileChallengeInfo, error) {
	m.initCertProgress(domain)
	m.addCertLog(domain, "info", fmt.Sprintf("开始文件验证申请: %s (提供商: %s)", domain, provider))
	m.setCertStep(domain, 1, "注册账户", "running")

	m.mu.Lock()
	acmeDir := m.acmeDir
	m.mu.Unlock()

	user, err := getOrLoadAccount(provider, email, acmeDir)
	if err != nil {
		return nil, err
	}

	eabKid, eabHmac := m.GetEABCredentials()
	client, err := newLegoClient(user, provider, eabKid, eabHmac)
	if err != nil {
		return nil, err
	}

	// 使用自定义 HTTP provider 获取 challenge 信息但不自动验证
	fileProvider := newFileProviderCapture()
	if err := client.Challenge.SetHTTP01Provider(fileProvider); err != nil {
		return nil, fmt.Errorf("set http01 provider: %w", err)
	}

	// 触发获取 authorization 以提取 challenge
	req := certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	}

	// 我们需要在 Present 阶段捕获 token 和 keyAuth
	// 先启动获取流程，fileProvider 会在 Present 时捕获
	go func() {
		_, _ = client.Certificate.Obtain(req)
	}()

	// 等待捕获
	info, err := fileProvider.WaitForChallenge(60 * time.Second)
	if err != nil {
		m.setCertStep(domain, 1, "注册账户", "error")
		m.addCertLog(domain, "error", "捕获验证信息失败: "+err.Error())
		return nil, err
	}

	m.setCertStep(domain, 1, "注册账户", "running")
	m.addCertLog(domain, "success", "账户注册完成")

	m.setCertStep(domain, 2, "准备验证", "running")
	m.addCertLog(domain, "success", "验证文件已生成，等待用户放置文件")

	m.setCertStep(domain, 3, "等待验证", "running")

	// 存储到 pendingChallenges
	m.mu.Lock()
	if m.pendingChallenges == nil {
		m.pendingChallenges = make(map[string]*PendingChallenge)
	}
	if m.acmeFileTokens == nil {
		m.acmeFileTokens = make(map[string]string)
	}
	m.pendingChallenges[domain] = &PendingChallenge{
		Domain:          domain,
		Email:           email,
		Provider:        provider,
		ChallengeMethod: "http-file",
		Client:          client,
		ExpiresAt:       time.Now().Add(30 * time.Minute),
	}
	m.acmeFileTokens[info.Token] = info.KeyAuth
	m.mu.Unlock()

	return &FileChallengeInfo{
		Token:     info.Token,
		KeyAuth:   info.KeyAuth,
		URLPath:   "/.well-known/acme-challenge/" + info.Token,
		VerifyURL: "http://" + domain + "/.well-known/acme-challenge/" + info.Token,
	}, nil
}

// CompleteFileChallenge 文件验证第二阶段：验证并获取证书
func (m *SSLManager) CompleteFileChallenge(domain string) error {
	m.mu.Lock()
	challenge, ok := m.pendingChallenges[domain]
	if !ok {
		m.mu.Unlock()
		m.setCertStep(domain, 3, "等待验证", "error")
		m.addCertLog(domain, "error", "没有找到待验证的文件挑战: "+domain)
		return fmt.Errorf("没有找到待验证的文件挑战: %s", domain)
	}
	if time.Now().After(challenge.ExpiresAt) {
		delete(m.pendingChallenges, domain)
		m.mu.Unlock()
		m.setCertStep(domain, 3, "等待验证", "error")
		m.addCertLog(domain, "error", "文件挑战已过期，请重新申请")
		return fmt.Errorf("文件挑战已过期，请重新申请")
	}
	certDir := m.certDir
	m.mu.Unlock()

	m.setCertStep(domain, 3, "等待验证", "running")
	m.addCertLog(domain, "info", "开始文件验证...")

	// 由于 fileProviderCapture 模式下 Obtain 已经启动，
	// 我们需要用新 client 重新获取（文件已经就位）
	user, err := getOrLoadAccount(challenge.Provider, challenge.Email, filepath.Join(m.certDir, "..", "ssl", "acme"))
	if err != nil {
		m.setCertStep(domain, 3, "等待验证", "error")
		m.addCertLog(domain, "error", "加载账户失败: "+err.Error())
		return err
	}
	_ = user // 重新创建 client

	// 使用自动托管验证文件的模式
	m.mu.RLock()
	acmeDir := m.acmeDir
	m.mu.RUnlock()

	user2, err := getOrLoadAccount(challenge.Provider, challenge.Email, acmeDir)
	if err != nil {
		m.setCertStep(domain, 3, "等待验证", "error")
		return err
	}
	eabKid, eabHmac := m.GetEABCredentials()
	client, err := newLegoClient(user2, challenge.Provider, eabKid, eabHmac)
	if err != nil {
		m.setCertStep(domain, 3, "等待验证", "error")
		return err
	}

	// 使用管理面板托管的 handler 作为 provider
	selfProvider := newSelfHostProvider(m)
	if err := client.Challenge.SetHTTP01Provider(selfProvider); err != nil {
		m.setCertStep(domain, 3, "等待验证", "error")
		return fmt.Errorf("set http01 provider: %w", err)
	}

	m.setCertStep(domain, 4, "获取证书", "running")
	m.addCertLog(domain, "info", "正在获取证书...")

	req := certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	}

	oldLogger := m.setLegoProgressLogger(domain)
	certResource, err := obtainWithTimeout(client, req, m, domain)
	restoreLegoLogger(oldLogger)
	if err != nil {
		return fmt.Errorf("obtain certificate: %w", err)
	}

	// 保存证书
	if err := m.saveLegoCertResource(domain, certResource, certDir); err != nil {
		m.setCertStep(domain, 4, "获取证书", "error")
		m.addCertLog(domain, "error", "保存证书失败: "+err.Error())
		return err
	}

	certPath := filepath.Join(certDir, domain+".crt")
	keyPath := filepath.Join(certDir, domain+".key")
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return fmt.Errorf("load cert: %w", err)
	}

	m.mu.Lock()
	m.certs[domain] = &cert
	if _, ok := m.configs[domain]; !ok {
		m.configs[domain] = &config.SSLConfig{Enabled: true}
	}
	cfg := m.configs[domain]
	cfg.Enabled = true
	cfg.AutoHTTPS = true
	cfg.Provider = challenge.Provider
	cfg.ChallengeMethod = "http-file"
	cfg.Email = challenge.Email
	m.updateCertInfoFromCert(domain, cfg, &cert)
	// 清理
	delete(m.pendingChallenges, domain)
	m.mu.Unlock()

	log.Printf("[SSL] 文件验证证书获取成功: %s", domain)
	m.setCertStep(domain, 5, "完成", "success")
	m.addCertLog(domain, "success", "文件验证证书获取成功")
	return nil
}

// PrepareDNSChallenge DNS 验证第一阶段：生成 TXT 记录信息或自动添加
func (m *SSLManager) PrepareDNSChallenge(domain, email, provider, dnsProviderName string, dnsCredentials map[string]string) (*DNSChallengeInfo, error) {
	m.initCertProgress(domain)
	m.addCertLog(domain, "info", fmt.Sprintf("开始 DNS 验证申请: %s (提供商: %s, DNS: %s)", domain, provider, dnsProviderName))
	m.setCertStep(domain, 1, "注册账户", "running")

	m.mu.Lock()
	acmeDir := m.acmeDir
	m.mu.Unlock()

	user, err := getOrLoadAccount(provider, email, acmeDir)
	if err != nil {
		return nil, err
	}

	eabKid, eabHmac := m.GetEABCredentials()
	client, err := newLegoClient(user, provider, eabKid, eabHmac)
	if err != nil {
		return nil, err
	}

	// 选择 DNS provider
	var dnsProvider challenge.Provider
	switch dnsProviderName {
	case "alidns":
		config := alidns.NewDefaultConfig()
		if v, ok := dnsCredentials["access_key"]; ok {
			config.APIKey = v
		}
		if v, ok := dnsCredentials["secret_key"]; ok {
			config.SecretKey = v
		}
		dnsProvider, err = alidns.NewDNSProviderConfig(config)
		if err != nil {
			return nil, fmt.Errorf("create alidns provider: %w", err)
		}
	case "tencentcloud":
		config := tencentcloud.NewDefaultConfig()
		if v, ok := dnsCredentials["secret_id"]; ok {
			config.SecretID = v
		}
		if v, ok := dnsCredentials["secret_key"]; ok {
			config.SecretKey = v
		}
		dnsProvider, err = tencentcloud.NewDNSProviderConfig(config)
		if err != nil {
			return nil, fmt.Errorf("create tencentcloud provider: %w", err)
		}
	case "baota":
		baotaConfig := &BaotaDNSConfig{}
		if v, ok := dnsCredentials["account_id"]; ok {
			baotaConfig.AccountID = v
		}
		if v, ok := dnsCredentials["access_key"]; ok {
			baotaConfig.AccessKey = v
		}
		if v, ok := dnsCredentials["secret_key"]; ok {
			baotaConfig.SecretKey = v
		}
		if v, ok := dnsCredentials["domain_id"]; ok {
			baotaConfig.DomainID = v
		}
		dnsProvider, err = NewBaotaDNSProvider(baotaConfig)
		if err != nil {
			return nil, fmt.Errorf("create baota provider: %w", err)
		}
	default:
		// 手动模式
		dnsProvider = newManualDNSProvider()
	}

	if err := client.Challenge.SetDNS01Provider(dnsProvider,
		dns01.AddRecursiveNameservers([]string{"8.8.8.8:53", "223.5.5.5:53"}),
	); err != nil {
		return nil, fmt.Errorf("set dns01 provider: %w", err)
	}

	// 对于手动模式，需要提取 TXT 记录信息给用户
	var dnsInfo *DNSChallengeInfo
	if dnsProviderName == "" || dnsProviderName == "manual" {
		// 使用 wrapped provider 捕获 challenge 信息
		wrapper := newCaptureDNSProvider()
		if err := client.Challenge.SetDNS01Provider(wrapper,
			dns01.AddRecursiveNameservers([]string{"8.8.8.8:53", "223.5.5.5:53"}),
		); err != nil {
			return nil, fmt.Errorf("set dns01 capture provider: %w", err)
		}

		// 异步启动申请流程
		go func() {
			req := certificate.ObtainRequest{Domains: []string{domain}, Bundle: true}
			_, _ = client.Certificate.Obtain(req)
		}()

		// 等待捕获 challenge 信息
		info, err := wrapper.WaitForChallenge(60 * time.Second)
		if err != nil {
			return nil, err
		}
		dnsInfo = info
	}

	// API 模式（alidns/tencentcloud/baota）：自动启动 Obtain
	isManual := dnsProviderName == "" || dnsProviderName == "manual"
	if !isManual {
		m.setCertStep(domain, 1, "注册账户", "done")
		m.addCertLog(domain, "success", "ACME 账户就绪")

		m.setCertStep(domain, 2, "准备验证", "running")
		m.addCertLog(domain, "info", "正在自动添加 DNS TXT 记录...")

		certDir := m.certDir

		go func() {
			// 捕获 lego 日志到进度系统
			oldLogger := m.setLegoProgressLogger(domain)
			defer restoreLegoLogger(oldLogger)

			m.setCertStep(domain, 3, "等待验证", "running")
			m.addCertLog(domain, "info", "DNS 记录已添加，等待 CA 验证（可能需要几分钟）...")

			req := certificate.ObtainRequest{Domains: []string{domain}, Bundle: true}
			certResource, err := obtainWithTimeout(client, req, m, domain)
			if err != nil {
				// obtainWithTimeout 已添加日志和步骤更新
				return
			}

			m.addCertLog(domain, "success", "证书获取成功，正在保存...")

			// 保存证书到磁盘
			if err := m.saveLegoCertResource(domain, certResource, certDir); err != nil {
				m.addCertLog(domain, "error", "保存证书失败: "+err.Error())
				m.setCertStep(domain, 4, "验证获取", "error")
				return
			}

			// 加载到内存
			certPath := filepath.Join(certDir, domain+".crt")
			keyPath := filepath.Join(certDir, domain+".key")
			cert, err := tls.LoadX509KeyPair(certPath, keyPath)
			if err != nil {
				m.addCertLog(domain, "error", "加载证书失败: "+err.Error())
				m.setCertStep(domain, 4, "验证获取", "error")
				return
			}

			m.mu.Lock()
			m.certs[domain] = &cert
			if _, ok := m.configs[domain]; !ok {
				m.configs[domain] = &config.SSLConfig{Enabled: true}
			}
			cfg := m.configs[domain]
			cfg.Enabled = true
			cfg.AutoHTTPS = true
			cfg.Provider = provider
			cfg.ChallengeMethod = "dns"
			cfg.Email = email
			m.updateCertInfoFromCert(domain, cfg, &cert)
			// 清理 pending challenge
			delete(m.pendingChallenges, domain)
			m.mu.Unlock()

			// 通知外部系统更新站点配置
			if m.onCertObtained != nil {
				m.onCertObtained(domain, provider, "dns", email)
			}

			log.Printf("[SSL] DNS 自动验证证书获取成功: %s", domain)
			m.setCertStep(domain, 5, "完成", "success")
			m.addCertLog(domain, "success", "DNS 验证证书获取成功！")
		}()

		return &DNSChallengeInfo{
			FQDN:       fmt.Sprintf("_acme-challenge.%s", strings.TrimPrefix(domain, "*.")),
			Value:      "(自动添加中...)",
			RecordType: "TXT",
		}, nil
	}

	// 手动模式：存储 pending challenge，等用户手动添加 TXT 记录
	m.mu.Lock()
	if m.pendingChallenges == nil {
		m.pendingChallenges = make(map[string]*PendingChallenge)
	}
	m.pendingChallenges[domain] = &PendingChallenge{
		Domain:          domain,
		Email:           email,
		Provider:        provider,
		ChallengeMethod: "dns",
		Client:          client,
		ExpiresAt:       time.Now().Add(30 * time.Minute),
		DNSProvider:     dnsProvider,
	}
	m.mu.Unlock()

	return dnsInfo, nil
}

// CompleteDNSChallenge DNS 验证第二阶段：用户确认后验证并获取证书
func (m *SSLManager) CompleteDNSChallenge(domain string) error {
	m.addCertLog(domain, "info", "用户已确认 DNS 记录，开始验证...")
	m.setCertStep(domain, 4, "验证获取", "running")

	m.mu.Lock()
	ch, ok := m.pendingChallenges[domain]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("没有找到待验证的 DNS 挑战: %s", domain)
	}
	if time.Now().After(ch.ExpiresAt) {
		delete(m.pendingChallenges, domain)
		m.mu.Unlock()
		return fmt.Errorf("DNS 挑战已过期，请重新申请")
	}
	certDir := m.certDir
	m.mu.Unlock()

	// 手动模式：需要重新获取（之前的 Obtain 因为 Present/CleanUp 是空操作会失败）
	// API 模式：如果 Present 成功了，Obtain 可能还在进行中
	// 简化方案：使用新的 client 重新发起 Obtain，DNS 记录已就位
	m.mu.RLock()
	acmeDir := m.acmeDir
	m.mu.RUnlock()

	user, err := getOrLoadAccount(ch.Provider, ch.Email, acmeDir)
	if err != nil {
		return err
	}
	eabKid, eabHmac := m.GetEABCredentials()
	client, err := newLegoClient(user, ch.Provider, eabKid, eabHmac)
	if err != nil {
		return err
	}

	// 设置 DNS provider（如果之前是 API 模式，记录已经存在）
	if ch.DNSProvider != nil {
		if err := client.Challenge.SetDNS01Provider(ch.DNSProvider,
			dns01.AddRecursiveNameservers([]string{"8.8.8.8:53", "223.5.5.5:53"}),
			dns01.WrapPreCheck(func(domain, fqdn, value string, check dns01.PreCheckFunc) (bool, error) {
				return true, nil // 跳过本地 DNS 传播检查
			}),
		); err != nil {
			return fmt.Errorf("set dns01 provider: %w", err)
		}
	} else {
		// 手动模式：使用 manualDNSProvider（Present 是空操作，CleanUp 是空操作）
		manual := newManualDNSProvider()
		if err := client.Challenge.SetDNS01Provider(manual,
			dns01.AddRecursiveNameservers([]string{"8.8.8.8:53", "223.5.5.5:53"}),
			dns01.WrapPreCheck(func(domain, fqdn, value string, check dns01.PreCheckFunc) (bool, error) {
				return true, nil // 跳过本地 DNS 传播检查
			}),
		); err != nil {
			return fmt.Errorf("set dns01 provider: %w", err)
		}
	}

	req := certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	}

	oldLogger := m.setLegoProgressLogger(domain)
	certResource, err := obtainWithTimeout(client, req, m, domain)
	restoreLegoLogger(oldLogger)
	if err != nil {
		return fmt.Errorf("obtain certificate: %w", err)
	}

	// 保存证书
	if err := m.saveLegoCertResource(domain, certResource, certDir); err != nil {
		return err
	}

	certPath := filepath.Join(certDir, domain+".crt")
	keyPath := filepath.Join(certDir, domain+".key")
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return fmt.Errorf("load cert: %w", err)
	}

	m.mu.Lock()
	m.certs[domain] = &cert
	if _, ok := m.configs[domain]; !ok {
		m.configs[domain] = &config.SSLConfig{Enabled: true}
	}
	cfg := m.configs[domain]
	cfg.Enabled = true
	cfg.AutoHTTPS = true
	cfg.Provider = ch.Provider
	cfg.ChallengeMethod = "dns"
	cfg.Email = ch.Email
	cfg.DNSProvider = "" // 不存储在站点配置中
	m.updateCertInfoFromCert(domain, cfg, &cert)
	delete(m.pendingChallenges, domain)
	m.mu.Unlock()

	log.Printf("[SSL] DNS 验证证书获取成功: %s", domain)
	m.setCertStep(domain, 5, "完成", "success")
	m.addCertLog(domain, "success", "DNS 验证证书获取成功")
	return nil
}

// GetACMEChallengeHandler 返回文件验证的 HTTP handler（注册到管理面板路由）
func (m *SSLManager) GetACMEChallengeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 提取 token
		token := strings.TrimPrefix(r.URL.Path, "/.well-known/acme-challenge/")
		if token == "" {
			http.NotFound(w, r)
			return
		}

		m.mu.RLock()
		keyAuth, ok := m.acmeFileTokens[token]
		m.mu.RUnlock()

		if !ok {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(keyAuth))
	}
}

// saveLegoCertResource 保存 lego 证书资源到磁盘
func (m *SSLManager) saveLegoCertResource(domain string, res *certificate.Resource, certDir string) error {
	if res == nil {
		return fmt.Errorf("certificate resource is nil")
	}

	certPath := filepath.Join(certDir, domain+".crt")
	keyPath := filepath.Join(certDir, domain+".key")

	if err := os.WriteFile(certPath, res.Certificate, 0644); err != nil {
		return fmt.Errorf("save certificate: %w", err)
	}
	if err := os.WriteFile(keyPath, res.PrivateKey, 0600); err != nil {
		os.Remove(certPath)
		return fmt.Errorf("save private key: %w", err)
	}

	return nil
}

// RenewCertificateLego 使用 lego 续期证书
func (m *SSLManager) RenewCertificateLego(domain string) error {
	m.mu.RLock()
	cfg, ok := m.configs[domain]
	acmeDir := m.acmeDir
	certDir := m.certDir
	m.mu.RUnlock()

	if !ok || cfg == nil {
		return fmt.Errorf("domain config not found: %s", domain)
	}

	user, err := getOrLoadAccount(cfg.GetProvider(), cfg.Email, acmeDir)
	if err != nil {
		return err
	}
	eabKid, eabHmac := m.GetEABCredentials()
	client, err := newLegoClient(user, cfg.GetProvider(), eabKid, eabHmac)
	if err != nil {
		return err
	}

	// 加载现有证书
	certPath := filepath.Join(certDir, domain+".crt")
	keyPath := filepath.Join(certDir, domain+".key")
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("read existing cert: %w", err)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("read existing key: %w", err)
	}

	// 续期（带超时保护）
	m.initCertProgress(domain)
	m.addCertLog(domain, "info", fmt.Sprintf("开始续期证书: %s", domain))
	m.setCertStep(domain, 4, "续期获取", "running")

	oldLogger := m.setLegoProgressLogger(domain)
	res, err := renewWithTimeout(client, certificate.Resource{
		Domain:      domain,
		Certificate: certPEM,
		PrivateKey:  keyPEM,
	}, m, domain)
	restoreLegoLogger(oldLogger)
	if err != nil {
		m.addCertLog(domain, "error", "证书续期失败: "+err.Error())
		m.setCertStep(domain, 4, "续期获取", "error")
		return fmt.Errorf("renew certificate: %w", err)
	}

	m.addCertLog(domain, "success", "证书续期成功，正在保存...")

	if err := m.saveLegoCertResource(domain, res, certDir); err != nil {
		return err
	}

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.certs[domain] = &cert
	m.updateCertInfoFromCert(domain, cfg, &cert)
	m.mu.Unlock()

	log.Printf("[SSL] lego 证书续期成功: %s", domain)
	return nil
}

// ========== HTTP Provider 辅助类型 ==========

// fileProviderCapture 捕获 HTTP-01 challenge 信息
type fileProviderCapture struct {
	ch chan *fileChallengeData
}

type fileChallengeData struct {
	Token   string
	KeyAuth string
}

func newFileProviderCapture() *fileProviderCapture {
	return &fileProviderCapture{ch: make(chan *fileChallengeData, 1)}
}

func (p *fileProviderCapture) Present(domain, token, keyAuth string) error {
	select {
	case p.ch <- &fileChallengeData{Token: token, KeyAuth: keyAuth}:
	default:
	}
	// 返回错误阻止 Obtain 继续自动验证
	return fmt.Errorf("waiting for user to place file")
}

func (p *fileProviderCapture) CleanUp(domain, token, keyAuth string) error {
	return nil
}

func (p *fileProviderCapture) WaitForChallenge(timeout time.Duration) (*FileChallengeInfo, error) {
	select {
	case data := <-p.ch:
		return &FileChallengeInfo{
			Token:   data.Token,
			KeyAuth: data.KeyAuth,
			URLPath: "/.well-known/acme-challenge/" + data.Token,
		}, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for challenge info")
	}
}

// selfHostProvider 使用 SSLManager.acmeFileTokens 作为验证数据源
type selfHostProvider struct {
	manager *SSLManager
}

func newSelfHostProvider(m *SSLManager) *selfHostProvider {
	return &selfHostProvider{manager: m}
}

func (p *selfHostProvider) Present(domain, token, keyAuth string) error {
	p.manager.mu.Lock()
	if p.manager.acmeFileTokens == nil {
		p.manager.acmeFileTokens = make(map[string]string)
	}
	p.manager.acmeFileTokens[token] = keyAuth
	p.manager.mu.Unlock()
	return nil
}

func (p *selfHostProvider) CleanUp(domain, token, keyAuth string) error {
	p.manager.mu.Lock()
	delete(p.manager.acmeFileTokens, token)
	p.manager.mu.Unlock()
	return nil
}

// captureDNSProvider 捕获 DNS-01 challenge 信息（用于手动模式）
type captureDNSProvider struct {
	ch chan *DNSChallengeInfo
}

func newCaptureDNSProvider() *captureDNSProvider {
	return &captureDNSProvider{ch: make(chan *DNSChallengeInfo, 1)}
}

func (p *captureDNSProvider) Present(domain, token, keyAuth string) error {
	info := getDNSChallengeInfo(domain, keyAuth)
	select {
	case p.ch <- info:
	default:
	}
	// 返回错误阻止自动验证流程
	return fmt.Errorf("waiting for user to add DNS record")
}

func (p *captureDNSProvider) CleanUp(domain, token, keyAuth string) error {
	return nil
}

func (p *captureDNSProvider) WaitForChallenge(timeout time.Duration) (*DNSChallengeInfo, error) {
	select {
	case info := <-p.ch:
		return info, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for DNS challenge info")
	}
}

// ========== Lego 日志捕获 ==========

// certProgressLogger 将 lego 内部日志路由到证书申请进度系统
type certProgressLogger struct {
	manager *SSLManager
	domain  string
	backup  legoLog.StdLogger
}

// detectLegoLevel 从 lego 日志前缀推断级别
func detectLegoLevel(msg string) string {
	if strings.Contains(msg, "[WARN]") {
		return "warn"
	}
	if strings.Contains(msg, "[ERROR]") || strings.Contains(msg, "[FATAL]") {
		return "error"
	}
	return "info"
}

// cleanLegoMsg 移除 lego 日志前缀使显示更简洁
func cleanLegoMsg(msg string) string {
	for _, prefix := range []string{"[INFO] ", "[WARN] ", "[ERROR] ", "[FATAL] "} {
		if strings.HasPrefix(msg, prefix) {
			return strings.TrimPrefix(msg, prefix)
		}
	}
	return msg
}

func (l *certProgressLogger) Printf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	level := detectLegoLevel(msg)
	l.manager.addCertLog(l.domain, level, cleanLegoMsg(msg))
	if l.backup != nil {
		l.backup.Printf(format, args...)
	}
}

func (l *certProgressLogger) Print(args ...any) {
	msg := fmt.Sprint(args...)
	l.manager.addCertLog(l.domain, "info", msg)
	if l.backup != nil {
		l.backup.Print(args...)
	}
}

func (l *certProgressLogger) Println(args ...any) {
	msg := fmt.Sprint(args...)
	l.manager.addCertLog(l.domain, "info", strings.TrimRight(msg, "\n"))
	if l.backup != nil {
		l.backup.Println(args...)
	}
}

func (l *certProgressLogger) Fatalf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	l.manager.addCertLog(l.domain, "error", cleanLegoMsg(msg))
	// 不调用 backup.Fatal（会 os.Exit），仅记录错误
}

func (l *certProgressLogger) Fatal(args ...any) {
	msg := fmt.Sprint(args...)
	l.manager.addCertLog(l.domain, "error", msg)
}

func (l *certProgressLogger) Fatalln(args ...any) {
	msg := fmt.Sprint(args...)
	l.manager.addCertLog(l.domain, "error", strings.TrimRight(msg, "\n"))
}

// setLegoProgressLogger 设置 lego 日志路由到进度系统
func (m *SSLManager) setLegoProgressLogger(domain string) legoLog.StdLogger {
	old := legoLog.Logger
	legoLog.Logger = &certProgressLogger{manager: m, domain: domain, backup: old}
	return old
}

// restoreLegoLogger 恢复 lego 日志
func restoreLegoLogger(old legoLog.StdLogger) {
	if old != nil {
		legoLog.Logger = old
	}
}

// obtainResult Obtain 调用结果
type obtainResult struct {
	resource *certificate.Resource
	err      error
}

// obtainWithTimeout 带超时的 Obtain 调用（5 分钟超时）
func obtainWithTimeout(client *lego.Client, req certificate.ObtainRequest, m *SSLManager, domain string) (*certificate.Resource, error) {
	resultCh := make(chan obtainResult, 1)

	go func() {
		resource, err := client.Certificate.Obtain(req)
		resultCh <- obtainResult{resource: resource, err: err}
	}()

	select {
	case result := <-resultCh:
		return result.resource, result.err
	case <-time.After(5 * time.Minute):
		m.addCertLog(domain, "error", "证书申请超时（5分钟），请检查网络连接后重试")
		m.setCertStep(domain, 4, "验证获取", "error")
		// 后台排空 Obtain 结果，避免 goroutine 泄漏
		go func() { <-resultCh }()
		return nil, fmt.Errorf("证书申请超时")
	}
}

// renewWithTimeout 带超时的 Renew 调用（5 分钟超时）
func renewWithTimeout(client *lego.Client, res certificate.Resource, m *SSLManager, domain string) (*certificate.Resource, error) {
	type renewResult struct {
		resource *certificate.Resource
		err      error
	}
	resultCh := make(chan renewResult, 1)

	go func() {
		r, err := client.Certificate.Renew(res, true, false, "")
		resultCh <- renewResult{resource: r, err: err}
	}()

	select {
	case result := <-resultCh:
		return result.resource, result.err
	case <-time.After(5 * time.Minute):
		m.addCertLog(domain, "error", "证书续期超时（5分钟），请检查网络连接后重试")
		m.setCertStep(domain, 4, "续期获取", "error")
		go func() { <-resultCh }()
		return nil, fmt.Errorf("证书续期超时")
	}
}

// ========== 宝塔 DNS Provider ==========

// BaotaDNSConfig 宝塔 DNS API 配置
type BaotaDNSConfig struct {
	AccountID string
	AccessKey string
	SecretKey string
	DomainID  string // 域名 ID
	BaseURL   string // API 基础 URL
}

// BaotaDNSProvider 宝塔域名 DNS provider
type BaotaDNSProvider struct {
	config *BaotaDNSConfig
	client *http.Client
}

// NewBaotaDNSProvider 创建宝塔 DNS provider
func NewBaotaDNSProvider(cfg *BaotaDNSConfig) (*BaotaDNSProvider, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://domains.bt.cn"
	}
	return &BaotaDNSProvider{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (p *BaotaDNSProvider) Present(domain, token, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)

	// 提取主机记录：_acme-challenge.example.com -> _acme-challenge
	fqdn := strings.TrimSuffix(info.FQDN, ".")
	parts := strings.SplitN(fqdn, ".", 2)
	record := parts[0]
	if len(parts) < 2 {
		record = "_acme-challenge"
	}

	body := map[string]interface{}{
		"domain_id":   p.config.DomainID,
		"domain_type": 2, // 外部域名
		"record":      record,
		"type":        "TXT",
		"value":       info.Value,
		"ttl":         600,
	}

	_, err := p.doRequest("POST", "/api/v1/dns/record/create", body)
	if err != nil {
		return fmt.Errorf("baota create dns record: %w", err)
	}

	// 等待 DNS 传播
	time.Sleep(5 * time.Second)
	return nil
}

func (p *BaotaDNSProvider) CleanUp(domain, token, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)

	// 查找并删除记录
	fqdn := strings.TrimSuffix(info.FQDN, ".")
	parts := strings.SplitN(fqdn, ".", 2)
	record := parts[0]

	// 查询记录列表
	listBody := map[string]interface{}{
		"domain_id":   p.config.DomainID,
		"domain_type": 2,
	}
	result, err := p.doRequest("POST", "/api/v1/dns/record/list", listBody)
	if err != nil {
		return nil // 清理失败不阻断
	}

	// 查找匹配的记录并删除
	if records, ok := result["data"].([]interface{}); ok {
		for _, r := range records {
			if rec, ok := r.(map[string]interface{}); ok {
				if rec["record"] == record && rec["type"] == "TXT" {
					if id, ok := rec["id"]; ok {
						delBody := map[string]interface{}{
							"domain_id": p.config.DomainID,
							"record_id": id,
						}
						p.doRequest("POST", "/api/v1/dns/record/delete", delBody)
					}
				}
			}
		}
	}

	return nil
}

// doRequest 发送宝塔 API 请求
func (p *BaotaDNSProvider) doRequest(method, path string, body interface{}) (map[string]interface{}, error) {
	var bodyBytes []byte
	var err error
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(context.Background(), method, p.config.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	if bodyBytes != nil {
		req.Body = http.NoBody
		req, err = http.NewRequestWithContext(context.Background(), method, p.config.BaseURL+path, strings.NewReader(string(bodyBytes)))
		if err != nil {
			return nil, err
		}
	}

	// 计算签名
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signingString := fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
		p.config.AccountID, timestamp, method, path, string(bodyBytes))

	// HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(p.config.SecretKey))
	mac.Write([]byte(signingString))
	signature := fmt.Sprintf("%x", mac.Sum(nil))

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Account-ID", p.config.AccountID)
	req.Header.Set("X-Access-Key", p.config.AccessKey)
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", signature)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return result, nil
}
