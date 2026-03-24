package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// ========== 兼容层：旧 Config 结构 ==========

// Config 旧配置结构（兼容）
type Config struct {
	HTTP     HTTPConfig     `json:"http"`
	FTP      FTPConfig      `json:"ftp"`
	Admin    AdminConfig    `json:"admin"`
	Global   GlobalConfig   `json:"global"`
	Sites    []SiteConfig   `json:"sites"`
	Log      LogConfig      `json:"log"`
}

// HTTPConfig HTTP 配置（兼容）
type HTTPConfig struct {
	Port   int    `json:"port"`
	Root   string `json:"root"`
	Domain string `json:"domain"`
}

// AdminConfig Admin 配置（兼容）
type AdminConfig struct {
	Enabled  bool   `json:"enabled"`
	Path     string `json:"path"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// GlobalConfig 全局配置（兼容）
type GlobalConfig struct {
	AdminPort int    `json:"admin_port"`
	FTPDir    string `json:"ftp_dir"`
	BackupDir string `json:"backup_dir"`
}

// Load 加载配置（兼容旧接口）
func Load(path string) (*Config, error) {
	// 尝试加载新的 config 目录
	cm, err := NewConfigManager("config")
	if err == nil {
		return convertToLegacyConfig(cm), nil
	}

	// 如果新配置不存在，尝试加载旧配置文件
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			return cfg, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// convertToLegacyConfig 将新配置转换为旧格式
func convertToLegacyConfig(cm *ConfigManager) *Config {
	cfg := &Config{
		Global: GlobalConfig{
			AdminPort: cm.Server.AdminPort,
			FTPDir:    cm.Server.FTPDir,
			BackupDir: cm.Server.BackupDir,
		},
		HTTP: HTTPConfig{
			Port:   cm.Server.HTTPPort,
			Root:   "./web",
			Domain: "localhost",
		},
		Admin: AdminConfig{
			Enabled:  true,
			Path:     cm.Server.AdminPath,
			Username: cm.Server.AdminUsername,
			Password: "", // 密码已加密
		},
		Log:   cm.Server.Log,
		Sites: cm.Sites.Sites,
		FTP:   *cm.FTP,
	}

	// 从第一个站点获取 HTTP 配置
	if len(cm.Sites.Sites) > 0 {
		site := cm.Sites.Sites[0]
		cfg.HTTP.Port = site.Port
		cfg.HTTP.Root = site.Root
		if len(site.Domain) > 0 {
			cfg.HTTP.Domain = site.Domain[0]
		}
	}

	return cfg
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		HTTP: HTTPConfig{
			Port:   8080,
			Root:   "./web",
			Domain: "localhost",
		},
		FTP: FTPConfig{
			Enabled: false,
			Port:    2121,
			Root:    "./ftp",
		},
		Admin: AdminConfig{
			Enabled:  true,
			Path:     "/admin",
			Username: "admin",
			Password: "admin123",
		},
		Global: GlobalConfig{
			AdminPort: 9527,
			FTPDir:    "./ftp",
			BackupDir: "./backups",
		},
		Sites: []SiteConfig{},
		Log: LogConfig{
			RetentionDays: 30,
			MaxSizeMB:     100,
			CompressDays:  7,
			CleanupHour:   3,
			Level:         "info",
		},
	}
}

// Save 保存配置（兼容）
func (c *Config) Save(path ...string) error {
	// 不再支持保存旧格式
	return fmt.Errorf("请使用新配置管理器")
}

// CreateDefaultDirectories 创建默认目录（兼容）
func (c *Config) CreateDefaultDirectories() error {
	return nil
}

// GetSiteByID 根据 ID 获取站点（兼容）
func (c *Config) GetSiteByID(id string) *SiteConfig {
	for i := range c.Sites {
		if c.Sites[i].ID == id {
			return &c.Sites[i]
		}
	}
	return nil
}

// AddSite 添加站点（兼容）
func (c *Config) AddSite(site SiteConfig) error {
	site.CreatedAt = time.Now().Format(time.RFC3339)
	site.UpdatedAt = time.Now().Format(time.RFC3339)
	c.Sites = append(c.Sites, site)
	return nil
}

// UpdateSite 更新站点（兼容）
func (c *Config) UpdateSite(id string, updated SiteConfig) bool {
	for i := range c.Sites {
		if c.Sites[i].ID == id {
			updated.ID = id
			updated.CreatedAt = c.Sites[i].CreatedAt
			updated.UpdatedAt = time.Now().Format(time.RFC3339)
			c.Sites[i] = updated
			return true
		}
	}
	return false
}

// DeleteSite 删除站点（兼容）
func (c *Config) DeleteSite(id string) bool {
	for i, site := range c.Sites {
		if site.ID == id {
			c.Sites = append(c.Sites[:i], c.Sites[i+1:]...)
			return true
		}
	}
	return false
}