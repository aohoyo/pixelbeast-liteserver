package config

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"pixelbeast/src/crypto"
)

// ConfigManager 配置管理器
type ConfigManager struct {
	mu sync.RWMutex

	configDir string
	key       []byte

	Server       *ServerConfig
	Sites        *SitesConfig
	FTP          *FTPConfig
	DNSProviders []DNSProviderConfig
}

// ServerConfig 服务配置
type ServerConfig struct {
	Name        string            `json:"name"`
	Timezone    string            `json:"timezone"`
	Admin       AdminConfig       `json:"admin"`
	Directories DirectoriesConfig `json:"directories"`
	Backup      BackupConfig      `json:"backup"`
	Log         LogConfig         `json:"log"`
	AutoStart   AutoStartConfig   `json:"auto_start"`
}

// AutoStartConfig 开机自启配置
type AutoStartConfig struct {
	Enabled bool `json:"enabled"` // 是否开机自启，默认 true
}

// DNSProviderConfig DNS 服务商配置（凭证加密存储）
type DNSProviderConfig struct {
	ID          string `json:"id"`          // 唯一标识
	Name        string `json:"name"`        // 显示名称 "阿里云 DNS"
	Type        string `json:"type"`        // "alidns" | "tencentcloud" | "baota"
	Credentials string `json:"credentials"` // AES-256-GCM 加密的凭证 JSON
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// AdminConfig 管理面板配置
type AdminConfig struct {
	Port                  int    `json:"port"`
	Username              string `json:"username"`
	Password              string `json:"password"`                // 加密后的密码
	Path                  string `json:"path"`                    // 安全入口路径
	Domain                string `json:"domain"`                  // 绑定域名，为空则允许所有域名访问
	SSLEnabled            bool   `json:"ssl_enabled"`
	RequirePasswordChange bool   `json:"require_password_change"` // 首次登录强制改密
}

// SitesConfig 站点配置列表
type SitesConfig struct {
	Sites []SiteConfig `json:"sites"`
}

// SiteConfig 站点配置
type SiteConfig struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"` // static, proxy

	// 静态站点
	Port       int      `json:"port"`
	Domain     []string `json:"domain"`
	Root       string   `json:"root,omitempty"`
	IndexFiles []string `json:"index_files"`
	AutoIndex  bool     `json:"auto_index"`

	// 反向代理
	Proxy *ProxyConfig `json:"proxy,omitempty"`

	// SSL
	SSL *SSLConfig `json:"ssl,omitempty"`

	// 元数据
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ProxyConfig 反向代理配置
type ProxyConfig struct {
	Target      string `json:"target"`
	StripPrefix string `json:"strip_prefix"`
	Websocket   bool   `json:"websocket"`
	Timeout     int    `json:"timeout"`
}

// SSLConfig SSL 配置
type SSLConfig struct {
	Enabled         bool   `json:"enabled"`
	AutoHTTPS       bool   `json:"auto_https"`
	Email           string `json:"email"`
	CertFile        string `json:"cert_file"`
	KeyFile         string `json:"key_file"`
	ForceHTTPS      bool   `json:"force_https"`
	HSTS            bool   `json:"hsts,omitempty"`             // Strict-Transport-Security
	Provider        string `json:"provider,omitempty"`         // "letsencrypt" | "litessl"
	ChallengeMethod string `json:"challenge_method,omitempty"` // "http-auto" | "http-file" | "dns"
	DNSProvider     string `json:"dns_provider,omitempty"`     // "manual" | "alidns" | "tencentcloud" | "baota"
	DNSCredentials  string `json:"dns_credentials,omitempty"`  // AES 加密的 DNS API 凭证 JSON
}

// IsAutoCert 是否使用自动证书（Let's Encrypt / LiteSSL HTTP-01）
func (s *SSLConfig) IsAutoCert() bool {
	return s != nil && s.Enabled && s.AutoHTTPS && s.ChallengeMethod != "dns"
}

// IsLegoCert 是否需要 lego 处理（非标准 autocert 场景）
func (s *SSLConfig) IsLegoCert() bool {
	if s == nil || !s.Enabled || !s.AutoHTTPS {
		return false
	}
	return s.Provider == "litessl" || s.ChallengeMethod == "http-file" || s.ChallengeMethod == "dns"
}

// GetProvider 获取证书提供商（默认 letsencrypt）
func (s *SSLConfig) GetProvider() string {
	if s == nil || s.Provider == "" {
		return "letsencrypt"
	}
	return s.Provider
}

// GetChallengeMethod 获取验证方式（默认 http-auto）
func (s *SSLConfig) GetChallengeMethod() string {
	if s == nil || s.ChallengeMethod == "" {
		return "http-auto"
	}
	return s.ChallengeMethod
}

// IsCustomCert 是否使用自定义证书
func (s *SSLConfig) IsCustomCert() bool {
	return s != nil && s.Enabled && !s.AutoHTTPS && s.CertFile != "" && s.KeyFile != ""
}

// FTPConfig FTP 配置
type FTPConfig struct {
	Enabled bool      `json:"enabled"`
	Port    int       `json:"port"`
	Users   []FTPUser `json:"users"`
}

// LogConfig 日志配置
type LogConfig struct {
	RetentionDays int               `json:"retention_days"`
	MaxSizeMB     int               `json:"max_size_mb"`
	CompressDays  int               `json:"compress_days"`
	CleanupHour   int               `json:"cleanup_hour"`
	Level         string            `json:"level"`
	Levels        map[string]string `json:"levels,omitempty"`
}

// DirectoriesConfig 目录配置
type DirectoriesConfig struct {
	Sites  string `json:"sites"`  // 站点默认根目录
	FTP    string `json:"ftp"`    // FTP 根目录
	Backup string `json:"backup"` // 备份目录
}

// BackupConfig 备份配置
type BackupConfig struct {
	AutoEnabled bool     `json:"auto_enabled"` // 自动备份
	Schedule    string   `json:"schedule"`     // daily, weekly, monthly
	Retention   int      `json:"retention"`    // 保留份数
	Items       []string `json:"items"`        // 备份内容：config, sites, ftp
}

// FTPUser FTP 用户
type FTPUser struct {
	Username       string `json:"username"`
	Password       string `json:"password"` // 加密后的密码
	RootPath       string `json:"root_path"`
	Status         string `json:"status"` // enabled, disabled
	Quota          int64  `json:"quota"`  // 容量限制（MB）
	UsedSpace      int64  `json:"used_space"`
	ExpiryDays     int    `json:"expiry_days"`
	ExpiryDate     string `json:"expiry_date"`
	Remark         string `json:"remark"`
	SpeedLimit     int64  `json:"speed_limit"`     // 下载速度限制 KB/s, 0=无限制
	Bandwidth      int64  `json:"bandwidth"`       // 上传速度限制 KB/s, 0=无限制
	MaxConnections int    `json:"max_connections"` // 最大连接数, 0=无限制
	MaxFiles       int    `json:"max_files"`       // 最大文件数量, 0=无限制
	MaxFileSize    int64  `json:"max_file_size"`   // 单文件大小限制 MB, 0=无限制
}

// NewConfigManager 创建配置管理器
func NewConfigManager(configDir string) (*ConfigManager, error) {
	cm := &ConfigManager{
		configDir: configDir,
	}

	// 确保目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 加载或创建密钥
	keyPath := filepath.Join(configDir, "secrets.key")
	key, err := crypto.EnsureKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("初始化密钥失败: %w", err)
	}
	cm.key = key

	// 加载配置
	if err := cm.load(); err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	return cm, nil
}

// load 加载所有配置
func (cm *ConfigManager) load() error {
	// 加载 server.json
	if err := cm.loadServer(); err != nil {
		return err
	}

	// 加载 sites.json
	if err := cm.loadSites(); err != nil {
		return err
	}

	// 加载 ftp.json
	if err := cm.loadFTP(); err != nil {
		return err
	}
	// 加载 dns_providers.json
	if err := cm.loadDNSProviders(); err != nil {
		return err
	}

	// 确保默认站点目录存在
	cm.ensureDefaultDirectories()

	return nil
}

// ensureDefaultDirectories 确保站点目录存在
func (cm *ConfigManager) ensureDefaultDirectories() {
	for _, site := range cm.Sites.Sites {
		if site.Type == "static" {
			if err := os.MkdirAll(cm.GetSiteRoot(&site), 0755); err != nil {
				fmt.Printf("[Config] 创建站点目录失败: %v\n", err)
			}
		}
	}
}

// loadOrCreate 加载配置文件，文件不存在时初始化默认值并保存
func (cm *ConfigManager) loadOrCreate(filename string, initDefaults func() error, unmarshal func(data []byte) error) error {
	path := filepath.Join(cm.configDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return initDefaults()
		}
		return err
	}
	return unmarshal(data)
}

// loadServer 加载服务配置
func (cm *ConfigManager) loadServer() error {
	return cm.loadOrCreate("server.json",
		func() error {
			cfg, err := cm.defaultServerConfig()
			if err != nil {
				return err
			}
			cm.Server = cfg
			return cm.saveServer()
		},
		func(data []byte) error {
			var cfg ServerConfig
			if err := json.Unmarshal(data, &cfg); err != nil {
				return err
			}
			cm.Server = &cfg
			cm.ensureDefaults()
			return nil
		},
	)
}

// ensureDefaults 确保必要字段有默认值
func (cm *ConfigManager) ensureDefaults() {
	changed := false

	if cm.Server.Directories.Sites == "" {
		cm.Server.Directories.Sites = "./sites"
		changed = true
	}
	if cm.Server.Directories.FTP == "" {
		cm.Server.Directories.FTP = "./ftp"
		changed = true
	}
	if cm.Server.Directories.Backup == "" {
		cm.Server.Directories.Backup = "./backups"
		changed = true
	}

	if cm.Server.Timezone == "" {
		cm.Server.Timezone = "Asia/Shanghai"
		changed = true
	}

	if cm.Server.Backup.Schedule == "" {
		cm.Server.Backup.Schedule = "daily"
		changed = true
	}
	if cm.Server.Backup.Items == nil {
		cm.Server.Backup.Items = []string{"config", "sites", "ftp"}
		changed = true
	}

	if changed {
		if err := cm.saveServer(); err != nil {
			fmt.Printf("[Config] 保存默认配置失败: %v\n", err)
		}
	}
}

// ========== 辅助方法 ==========

// ConfigDir 获取配置目录路径
func (cm *ConfigManager) ConfigDir() string {
	return cm.configDir
}

// GetFTPRoot 获取 FTP 根目录
func (cm *ConfigManager) GetFTPRoot() string {
	if cm.Server.Directories.FTP != "" {
		return cm.Server.Directories.FTP
	}
	return "./ftp"
}

// GetSiteRoot 获取站点根目录（优先使用自定义路径，否则默认 SitesDir/site.ID）
func (cm *ConfigManager) GetSiteRoot(site *SiteConfig) string {
	if site.Root != "" {
		return site.Root
	}
	return filepath.Join(cm.GetSitesDir(), site.ID)
}

// GetSitesDir 获取站点默认根目录
func (cm *ConfigManager) GetSitesDir() string {
	if cm.Server.Directories.Sites != "" {
		return cm.Server.Directories.Sites
	}
	return "./sites"
}

// GetBackupDir 获取备份目录
func (cm *ConfigManager) GetBackupDir() string {
	if cm.Server.Directories.Backup != "" {
		return cm.Server.Directories.Backup
	}
	return "./backups"
}

// loadSites 加载站点配置
func (cm *ConfigManager) loadSites() error {
	return cm.loadOrCreate("sites.json",
		func() error {
			cm.Sites = cm.defaultSitesConfig()
			return cm.saveSites()
		},
		func(data []byte) error {
			var cfg SitesConfig
			if err := json.Unmarshal(data, &cfg); err != nil {
				return err
			}
			cm.Sites = &cfg
			return nil
		},
	)
}

// loadFTP 加载 FTP 配置
func (cm *ConfigManager) loadFTP() error {
	return cm.loadOrCreate("ftp.json",
		func() error {
			cm.FTP = cm.defaultFTPConfig()
			return cm.saveFTP()
		},
		func(data []byte) error {
			var cfg FTPConfig
			if err := json.Unmarshal(data, &cfg); err != nil {
				return err
			}
			cm.FTP = &cfg
			return nil
		},
	)
}

// Save 保存所有配置
func (cm *ConfigManager) Save() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if err := cm.saveServer(); err != nil {
		return err
	}
	if err := cm.saveSites(); err != nil {
		return err
	}
	if err := cm.saveFTP(); err != nil {
		return err
	}
	if err := cm.saveDNSProviders(); err != nil {
		return err
	}
	return nil
}

// saveJSON 保存 JSON 配置文件
func (cm *ConfigManager) saveJSON(filename string, data interface{}) error {
	path := filepath.Join(cm.configDir, filename)
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0600)
}

// saveServer 保存服务配置
func (cm *ConfigManager) saveServer() error {
	return cm.saveJSON("server.json", cm.Server)
}

// saveSites 保存站点配置
func (cm *ConfigManager) saveSites() error {
	return cm.saveJSON("sites.json", cm.Sites)
}

// saveFTP 保存 FTP 配置
func (cm *ConfigManager) saveFTP() error {
	return cm.saveJSON("ftp.json", cm.FTP)
}

// loadDNSProviders 加载 DNS 服务商配置
func (cm *ConfigManager) loadDNSProviders() error {
	path := filepath.Join(cm.configDir, "dns_providers.json")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		var providers []DNSProviderConfig
		if err := json.Unmarshal(data, &providers); err != nil {
			return err
		}
		cm.DNSProviders = providers
		return nil
	}

	// dns_providers.json 不存在，尝试从 server.json 迁移旧数据
	serverPath := filepath.Join(cm.configDir, "server.json")
	serverData, readErr := os.ReadFile(serverPath)
	if readErr == nil {
		var raw map[string]json.RawMessage
		if json.Unmarshal(serverData, &raw) == nil {
			if oldField, ok := raw["dns_providers"]; ok {
				var oldProviders []DNSProviderConfig
				if json.Unmarshal(oldField, &oldProviders) == nil {
					cm.DNSProviders = oldProviders
				}
			}
			// 从 server.json 中移除 dns_providers 字段
			delete(raw, "dns_providers")
			newData, marshalErr := json.MarshalIndent(raw, "", "  ")
			if marshalErr != nil {
				return fmt.Errorf("序列化服务器配置失败: %w", marshalErr)
			}
			if writeErr := os.WriteFile(serverPath, newData, 0600); writeErr != nil {
				return writeErr
			}
		}
	}
	if cm.DNSProviders == nil {
		cm.DNSProviders = []DNSProviderConfig{}
	}
	return cm.saveDNSProviders()
}

// saveDNSProviders 保存 DNS 服务商配置
func (cm *ConfigManager) saveDNSProviders() error {
	providers := cm.DNSProviders
	if providers == nil {
		providers = []DNSProviderConfig{}
	}
	return cm.saveJSON("dns_providers.json", providers)
}

// defaultServerConfig 默认服务配置
func (cm *ConfigManager) defaultServerConfig() (*ServerConfig, error) {
	// 生成随机密码
	randomPassword, err := generateRandomPassword()
	if err != nil {
		return nil, fmt.Errorf("生成随机密码失败: %w", err)
	}

	encryptedPassword, err := crypto.EncryptString(randomPassword, cm.key)
	if err != nil {
		return nil, fmt.Errorf("加密默认密码失败: %w", err)
	}

	// 保存初始密码到文件
	if err := cm.saveInitialPassword(randomPassword); err != nil {
		return nil, fmt.Errorf("保存初始密码失败: %w", err)
	}

	return &ServerConfig{
		Name:     "PixelBeast Server",
		Timezone: "Asia/Shanghai",
		Admin: AdminConfig{
			Port:                  9527,
			Username:              "admin",
			Password:              encryptedPassword,
			Path:                  "/admin",
			RequirePasswordChange: true,
		},
		Directories: DirectoriesConfig{
			Sites:  "./sites",
			FTP:    "./ftp",
			Backup: "./backups",
		},
		Backup: BackupConfig{
			AutoEnabled: true,
			Schedule:    "daily",
			Retention:   3,
			Items:       []string{"config", "sites", "ftp"},
		},
		Log: LogConfig{
			RetentionDays: 30,
			MaxSizeMB:     100,
			CompressDays:  7,
			CleanupHour:   3,
			Level:         "info",
		},
		AutoStart: AutoStartConfig{
			Enabled: true,
		},
	}, nil
}

// defaultFTPConfig 默认 FTP 配置
func (cm *ConfigManager) defaultFTPConfig() *FTPConfig {
	return &FTPConfig{
		Enabled: false,
		Port:    2121,
		Users:   []FTPUser{},
	}
}

// defaultSitesConfig 默认站点配置
func (cm *ConfigManager) defaultSitesConfig() *SitesConfig {
	cfg := &SitesConfig{
		Sites: []SiteConfig{
			{
				ID:         "default",
				Name:       "默认站点",
				Enabled:    true,
				Type:       "static",
				Port:       3380,
				IndexFiles: []string{"index.html", "index.htm"},
				AutoIndex:  true,
				CreatedAt:  time.Now().Format("2006-01-02 15:04:05"),
			},
		},
	}
	return cfg
}

// ========== 配置重置 ==========

// ResetToDefaults 重置服务配置为默认值（保留站点和FTP用户数据）
func (cm *ConfigManager) ResetToDefaults() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cfg, err := cm.defaultServerConfig()
	if err != nil {
		return err
	}
	cm.Server = cfg
	return nil
}

// ========== Admin 密码管理 ==========

// GetSharedPort 获取共享端口（从第一个站点推导）
func (cm *ConfigManager) GetSharedPort() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for _, site := range cm.Sites.Sites {
		if site.Enabled && site.Port > 0 {
			return site.Port
		}
	}
	return 3380
}

// SetAdminPassword 设置管理员密码（加密存储）
func (cm *ConfigManager) SetAdminPassword(password string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	encrypted, err := crypto.EncryptString(password, cm.key)
	if err != nil {
		return err
	}

	cm.Server.Admin.Password = encrypted
	return cm.saveServer()
}

// GetAdminPassword 获取管理员密码（解密）
func (cm *ConfigManager) GetAdminPassword() (string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return crypto.DecryptString(cm.Server.Admin.Password, cm.key)
}

// GetKey 获取加密密钥（用于密码加密）
func (cm *ConfigManager) GetKey() []byte {
	return cm.key
}

// ValidateAdmin 验证管理员账号密码
func (cm *ConfigManager) ValidateAdmin(username, password string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.Server.Admin.Username != username {
		return false
	}

	decrypted, err := crypto.DecryptString(cm.Server.Admin.Password, cm.key)
	if err != nil {
		return false
	}

	return decrypted == password
}

// ========== FTP 用户密码管理 ==========

// SetFTPUserPassword 设置 FTP 用户密码（加密存储）
func (cm *ConfigManager) SetFTPUserPassword(username, password string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	encrypted, err := crypto.EncryptString(password, cm.key)
	if err != nil {
		return err
	}

	for i, user := range cm.FTP.Users {
		if user.Username == username {
			cm.FTP.Users[i].Password = encrypted
			return cm.saveFTP()
		}
	}

	return fmt.Errorf("用户不存在: %s", username)
}

// GetFTPUserPassword 获取 FTP 用户密码（解密）
func (cm *ConfigManager) GetFTPUserPassword(username string) (string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for _, user := range cm.FTP.Users {
		if user.Username == username {
			return crypto.DecryptString(user.Password, cm.key)
		}
	}

	return "", fmt.Errorf("用户不存在: %s", username)
}

// ValidateFTPUser 验证 FTP 用户密码
func (cm *ConfigManager) ValidateFTPUser(username, password string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for _, user := range cm.FTP.Users {
		if user.Username == username && user.Status == "enabled" {
			decrypted, err := crypto.DecryptString(user.Password, cm.key)
			if err != nil {
				return false
			}
			return decrypted == password
		}
	}

	return false
}

// GetFTPUserConfig 获取 FTP 用户配置（带锁，返回副本）
func (cm *ConfigManager) GetFTPUserConfig(username string) *FTPUser {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for _, user := range cm.FTP.Users {
		if user.Username == username && user.Status == "enabled" {
			cp := user
			return &cp
		}
	}
	return nil
}

// EncryptPassword 加密密码
func (cm *ConfigManager) EncryptPassword(password string) (string, error) {
	return crypto.EncryptString(password, cm.key)
}

// ========== 站点管理 ==========

// GetSiteByID 根据 ID 获取站点
func (cm *ConfigManager) GetSiteByID(id string) *SiteConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for i := range cm.Sites.Sites {
		if cm.Sites.Sites[i].ID == id {
			return &cm.Sites.Sites[i]
		}
	}
	return nil
}

// AddSite 添加站点
func (cm *ConfigManager) AddSite(site SiteConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 检查 ID 是否已存在
	for _, s := range cm.Sites.Sites {
		if s.ID == site.ID {
			return fmt.Errorf("站点 ID 已存在: %s", site.ID)
		}
	}

	site.CreatedAt = time.Now().Format(time.RFC3339)
	site.UpdatedAt = time.Now().Format(time.RFC3339)
	cm.Sites.Sites = append(cm.Sites.Sites, site)

	return cm.saveSites()
}

// UpdateSite 更新站点
func (cm *ConfigManager) UpdateSite(id string, updated SiteConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for i := range cm.Sites.Sites {
		if cm.Sites.Sites[i].ID == id {
			updated.ID = id
			updated.CreatedAt = cm.Sites.Sites[i].CreatedAt
			updated.UpdatedAt = time.Now().Format(time.RFC3339)
			cm.Sites.Sites[i] = updated
			return cm.saveSites()
		}
	}

	return fmt.Errorf("站点不存在: %s", id)
}

// deleteFromSlice 从切片中删除匹配元素并保存配置
func deleteFromSlice[T any](slice *[]T, match func(T) bool, save func() error, notFoundMsg string) error {
	for i, item := range *slice {
		if match(item) {
			*slice = append((*slice)[:i], (*slice)[i+1:]...)
			return save()
		}
	}
	return errors.New(notFoundMsg)
}

// DeleteSite 删除站点
func (cm *ConfigManager) DeleteSite(id string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return deleteFromSlice(&cm.Sites.Sites,
		func(s SiteConfig) bool { return s.ID == id },
		cm.saveSites,
		"站点不存在: "+id,
	)
}

// ========== FTP 用户管理 ==========

// GetFTPUser 获取 FTP 用户
func (cm *ConfigManager) GetFTPUser(username string) *FTPUser {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for i := range cm.FTP.Users {
		if cm.FTP.Users[i].Username == username {
			return &cm.FTP.Users[i]
		}
	}
	return nil
}

// AddFTPUser 添加 FTP 用户
func (cm *ConfigManager) AddFTPUser(user FTPUser, plainPassword string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 检查用户名是否已存在
	for _, u := range cm.FTP.Users {
		if u.Username == user.Username {
			return fmt.Errorf("用户名已存在: %s", user.Username)
		}
	}

	// 加密密码
	encrypted, err := crypto.EncryptString(plainPassword, cm.key)
	if err != nil {
		return err
	}
	user.Password = encrypted

	if user.Status == "" {
		user.Status = "enabled"
	}

	cm.FTP.Users = append(cm.FTP.Users, user)
	return cm.saveFTP()
}

// UpdateFTPUser 更新 FTP 用户
func (cm *ConfigManager) UpdateFTPUser(username string, updated FTPUser) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for i := range cm.FTP.Users {
		if cm.FTP.Users[i].Username == username {
			// 保留原密码，除非明确要更新
			updated.Password = cm.FTP.Users[i].Password
			updated.Username = username
			cm.FTP.Users[i] = updated
			return cm.saveFTP()
		}
	}

	return fmt.Errorf("用户不存在: %s", username)
}

// DeleteFTPUser 删除 FTP 用户
func (cm *ConfigManager) DeleteFTPUser(username string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return deleteFromSlice(&cm.FTP.Users,
		func(u FTPUser) bool { return u.Username == username },
		cm.saveFTP,
		"用户不存在: "+username,
	)
}

// ========== SSL 辅助 ==========

// GetSSLEnabledSites 获取所有启用了 SSL 的站点
func (cm *ConfigManager) GetSSLEnabledSites() []*SiteConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	var sites []*SiteConfig
	for i := range cm.Sites.Sites {
		if cm.Sites.Sites[i].SSL != nil && cm.Sites.Sites[i].SSL.Enabled {
			sites = append(sites, &cm.Sites.Sites[i])
		}
	}
	return sites
}

// ========== DNS 服务商管理 ==========

// GetDNSProviders 获取所有 DNS 服务商配置
func (cm *ConfigManager) GetDNSProviders() []DNSProviderConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	if cm.DNSProviders == nil {
		return []DNSProviderConfig{}
	}
	return append([]DNSProviderConfig{}, cm.DNSProviders...)
}

// GetDNSProvider 根据 ID 获取 DNS 服务商配置
func (cm *ConfigManager) GetDNSProvider(id string) *DNSProviderConfig {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	for i := range cm.DNSProviders {
		if cm.DNSProviders[i].ID == id {
			cp := cm.DNSProviders[i]
			return &cp
		}
	}
	return nil
}

// AddDNSProvider 添加 DNS 服务商配置
func (cm *ConfigManager) AddDNSProvider(provider DNSProviderConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for _, p := range cm.DNSProviders {
		if p.ID == provider.ID {
			return fmt.Errorf("DNS 服务商 ID 已存在: %s", provider.ID)
		}
	}
	cm.DNSProviders = append(cm.DNSProviders, provider)
	return cm.saveDNSProviders()
}

// UpdateDNSProvider 更新 DNS 服务商配置
func (cm *ConfigManager) UpdateDNSProvider(id string, updated DNSProviderConfig) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for i := range cm.DNSProviders {
		if cm.DNSProviders[i].ID == id {
			updated.ID = id
			cm.DNSProviders[i] = updated
			return cm.saveDNSProviders()
		}
	}
	return fmt.Errorf("DNS 服务商不存在: %s", id)
}

// DeleteDNSProvider 删除 DNS 服务商配置
func (cm *ConfigManager) DeleteDNSProvider(id string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	return deleteFromSlice(&cm.DNSProviders,
		func(p DNSProviderConfig) bool { return p.ID == id },
		cm.saveDNSProviders,
		"DNS 服务商不存在: "+id,
	)
}

// EncryptDNSCredentials 加密 DNS 凭证
func (cm *ConfigManager) EncryptDNSCredentials(creds map[string]string) (string, error) {
	jsonData, err := json.Marshal(creds)
	if err != nil {
		return "", fmt.Errorf("序列化凭证失败: %w", err)
	}
	return crypto.EncryptString(string(jsonData), cm.key)
}

// DecryptDNSCredentials 解密 DNS 凭证
func (cm *ConfigManager) DecryptDNSCredentials(encrypted string) (map[string]string, error) {
	plain, err := crypto.DecryptString(encrypted, cm.key)
	if err != nil {
		return nil, fmt.Errorf("解密凭证失败: %w", err)
	}
	var creds map[string]string
	if err := json.Unmarshal([]byte(plain), &creds); err != nil {
		return nil, fmt.Errorf("解析凭证失败: %w", err)
	}
	return creds, nil
}

// generateRandomPassword 生成随机密码（16位，包含大小写字母和数字）
func generateRandomPassword() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 16

	password := make([]byte, length)
	for i := range password {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		password[i] = charset[num.Int64()]
	}
	return string(password), nil
}

// saveInitialPassword 保存初始密码到临时文件（仅首次启动）
func (cm *ConfigManager) saveInitialPassword(password string) error {
	passwordFile := filepath.Join(cm.configDir, "initial_password.txt")

	content := fmt.Sprintf(`PixelBeast 初始密码
====================
账号：admin
密码：%s

⚠  重要提示:
1. 请在首次登录后立即修改密码
2. 此文件将在登录后自动删除
3. 如已修改密码，请手动删除此文件

生成时间：%s
`,
		password,
		time.Now().Format("2006-01-02 15:04:05"),
	)

	if err := os.WriteFile(passwordFile, []byte(content), 0600); err != nil {
		return err
	}

	return nil
}
