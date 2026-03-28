package config

import (
	"encoding/json"
	"fmt"
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

	Server *ServerConfig
	Sites  *SitesConfig
	FTP    *FTPConfig
}

// ServerConfig 服务配置
type ServerConfig struct {
	// HTTP
	HTTPPort  int `json:"http_port"`
	AdminPort int `json:"admin_port"`

	// Admin（密码加密存储）
	AdminUsername string `json:"admin_username"`
	AdminPassword string `json:"admin_password"` // 加密后的密码
	AdminPath     string `json:"admin_path"`     // 安全入口路径

	// 日志
	Log LogConfig `json:"log"`

	// 全局
	FTPDir    string `json:"ftp_dir"`
	BackupDir string `json:"backup_dir"`
}

// SitesConfig 站点配置
type SitesConfig struct {
	Sites []SiteConfig `json:"sites"`
}

// SiteConfig 站点配置
type SiteConfig struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Enabled   bool     `json:"enabled"`
	Type      string   `json:"type"` // static, proxy

	// 静态站点
	Port       int      `json:"port"`
	Domain     []string `json:"domain"`
	Root       string   `json:"root"`
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
	Enabled    bool   `json:"enabled"`
	AutoHTTPS  bool   `json:"auto_https"`
	Email      string `json:"email"`
	CertFile   string `json:"cert_file"`
	KeyFile    string `json:"key_file"`
	ForceHTTPS bool   `json:"force_https"`
}

// FTPConfig FTP 配置
type FTPConfig struct {
	Enabled bool      `json:"enabled"`
	Port    int       `json:"port"`
	Root    string    `json:"root"`
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

// FTPUser FTP 用户
type FTPUser struct {
	Username  string `json:"username"`
	Password  string `json:"password"` // 加密后的密码
	RootPath  string `json:"root_path"`
	Status    string `json:"status"` // enabled, disabled
	Quota     int64  `json:"quota"`  // 容量限制（MB）
	UsedSpace int64  `json:"used_space"`
	ExpiryDays int   `json:"expiry_days"`
	ExpiryDate string `json:"expiry_date"`
	Remark    string `json:"remark"`
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

	return nil
}

// loadServer 加载服务配置
func (cm *ConfigManager) loadServer() error {
	path := filepath.Join(cm.configDir, "server.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// 创建默认配置
			cm.Server = cm.defaultServerConfig()
			return cm.saveServer()
		}
		return err
	}

	var cfg ServerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	cm.Server = &cfg
	return nil
}

// loadSites 加载站点配置
func (cm *ConfigManager) loadSites() error {
	path := filepath.Join(cm.configDir, "sites.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cm.Sites = &SitesConfig{Sites: []SiteConfig{}}
			return cm.saveSites()
		}
		return err
	}

	var cfg SitesConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	cm.Sites = &cfg
	return nil
}

// loadFTP 加载 FTP 配置
func (cm *ConfigManager) loadFTP() error {
	path := filepath.Join(cm.configDir, "ftp.json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cm.FTP = cm.defaultFTPConfig()
			return cm.saveFTP()
		}
		return err
	}

	var cfg FTPConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	cm.FTP = &cfg
	return nil
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
	return nil
}

// saveServer 保存服务配置
func (cm *ConfigManager) saveServer() error {
	path := filepath.Join(cm.configDir, "server.json")
	data, err := json.MarshalIndent(cm.Server, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// saveSites 保存站点配置
func (cm *ConfigManager) saveSites() error {
	path := filepath.Join(cm.configDir, "sites.json")
	data, err := json.MarshalIndent(cm.Sites, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// saveFTP 保存 FTP 配置
func (cm *ConfigManager) saveFTP() error {
	path := filepath.Join(cm.configDir, "ftp.json")
	data, err := json.MarshalIndent(cm.FTP, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// defaultServerConfig 默认服务配置
func (cm *ConfigManager) defaultServerConfig() *ServerConfig {
	// 生成加密的默认密码
	encryptedPassword, _ := crypto.EncryptString("admin123", cm.key)

	return &ServerConfig{
		HTTPPort:      8080,
		AdminPort:     9527,
		AdminUsername: "admin",
		AdminPassword: encryptedPassword,
		AdminPath:     "/admin", // 默认安全入口
		Log: LogConfig{
			RetentionDays: 30,
			MaxSizeMB:     100,
			CompressDays:  7,
			CleanupHour:   3,
			Level:         "info",
		},
		FTPDir:    "./ftp",
		BackupDir: "./backups",
	}
}

// defaultFTPConfig 默认 FTP 配置
func (cm *ConfigManager) defaultFTPConfig() *FTPConfig {
	return &FTPConfig{
		Enabled: false,
		Port:    2121,
		Root:    "./ftp",
		Users:   []FTPUser{},
	}
}

// ========== Admin 密码管理 ==========

// SetAdminPassword 设置管理员密码（加密存储）
func (cm *ConfigManager) SetAdminPassword(password string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	encrypted, err := crypto.EncryptString(password, cm.key)
	if err != nil {
		return err
	}

	cm.Server.AdminPassword = encrypted
	return cm.saveServer()
}

// GetAdminPassword 获取管理员密码（解密）
func (cm *ConfigManager) GetAdminPassword() (string, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return crypto.DecryptString(cm.Server.AdminPassword, cm.key)
}

// GetKey 获取加密密钥（用于密码加密）
func (cm *ConfigManager) GetKey() []byte {
	return cm.key
}

// ValidateAdmin 验证管理员账号密码
func (cm *ConfigManager) ValidateAdmin(username, password string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.Server.AdminUsername != username {
		return false
	}

	decrypted, err := crypto.DecryptString(cm.Server.AdminPassword, cm.key)
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

// DeleteSite 删除站点
func (cm *ConfigManager) DeleteSite(id string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for i, site := range cm.Sites.Sites {
		if site.ID == id {
			cm.Sites.Sites = append(cm.Sites.Sites[:i], cm.Sites.Sites[i+1:]...)
			return cm.saveSites()
		}
	}

	return fmt.Errorf("站点不存在: %s", id)
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

	for i, user := range cm.FTP.Users {
		if user.Username == username {
			cm.FTP.Users = append(cm.FTP.Users[:i], cm.FTP.Users[i+1:]...)
			return cm.saveFTP()
		}
	}

	return fmt.Errorf("用户不存在: %s", username)
}