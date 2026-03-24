package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Config 主配置结构
type Config struct {
	// Legacy - 向后兼容，迁移后使用 Sites
	HTTP  HTTPConfig  `json:"http,omitempty"`
	FTP   FTPConfig   `json:"ftp"`
	Admin AdminConfig `json:"admin"`

	// New - 多站点配置
	Global GlobalConfig `json:"global,omitempty"`
	Sites  []SiteConfig `json:"sites,omitempty"`
}

// GlobalConfig 全局配置
type GlobalConfig struct {
	AdminPort int    `json:"admin_port"` // 管理面板独立端口
	DataDir   string `json:"data_dir"`   // 数据目录
}

// SiteConfig 站点配置
type SiteConfig struct {
	ID        string      `json:"id"`        // 唯一标识
	Name      string      `json:"name"`      // 站点名称
	Enabled   bool        `json:"enabled"`   // 是否启用
	Type      string      `json:"type"`      // 站点类型: static, proxy

	// 静态站点配置
	Port       int      `json:"port"`       // 端口 (0 = 共享端口)
	Domain     []string `json:"domain"`     // 域名列表
	Root       string   `json:"root"`       // 根目录 (static 类型)
	IndexFiles []string `json:"index_files"` // 默认索引文件
	AutoIndex  bool     `json:"auto_index"` // 是否显示目录列表

	// 反向代理配置
	Proxy *ProxyConfig `json:"proxy,omitempty"` // 反向代理配置 (proxy 类型)

	// SSL 配置
	SSL *SSLConfig `json:"ssl,omitempty"` // SSL 配置

	// 元数据
	CreatedAt string `json:"created_at"` // 创建时间
	UpdatedAt string `json:"updated_at"` // 更新时间
}

// ProxyConfig 反向代理配置
type ProxyConfig struct {
	Target      string `json:"target"`       // 目标 URL
	StripPrefix string `json:"strip_prefix"` // 要移除的前缀
	Websocket   bool   `json:"websocket"`    // 是否支持 WebSocket
	Timeout     int    `json:"timeout"`      // 超时时间（秒）
}

// SSLConfig SSL 配置
type SSLConfig struct {
	Enabled    bool   `json:"enabled"`     // 是否启用 SSL
	AutoHTTPS  bool   `json:"auto_https"`  // 是否自动申请证书
	Email      string `json:"email"`       // 证书联系邮箱
	CertFile   string `json:"cert_file"`   // 自定义证书路径
	KeyFile    string `json:"key_file"`    // 自定义私钥路径
	ForceHTTPS bool   `json:"force_https"` // 是否强制 HTTPS
}

// HTTPConfig HTTP服务器配置 (Legacy)
type HTTPConfig struct {
	Port   int    `json:"port"`
	Root   string `json:"root"`
	Domain string `json:"domain"`
}

// FTPConfig FTP服务器配置
type FTPConfig struct {
	Enabled bool      `json:"enabled"`
	Port    int       `json:"port"`
	Root    string    `json:"root"`
	Users   []FTPUser `json:"users"`
}

// FTPUser FTP用户
type FTPUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AdminConfig 安全入口配置
type AdminConfig struct {
	Enabled  bool       `json:"enabled"`
	Path     string     `json:"path"`
	Username string     `json:"username"`
	Password string     `json:"password"`
	SSL      *SSLConfig `json:"ssl,omitempty"` // 管理面板 SSL 配置
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		HTTP: HTTPConfig{
			Port:   8080,
			Root:   "./web",
			Domain: "",
		},
		FTP: FTPConfig{
			Enabled: false,
			Port:    2121,
			Root:    "./ftp",
			Users: []FTPUser{
				{Username: "flash", Password: "flash"},
			},
		},
		Admin: AdminConfig{
			Enabled:  true,
			Path:     "/admin",
			Username: "admin",
			Password: "admin123",
		},
		Global: GlobalConfig{
			AdminPort: 9527,
			DataDir:   "./data",
		},
		Sites: []SiteConfig{
			{
				ID:        "default",
				Name:      "默认网站",
				Enabled:   true,
				Type:      "static",
				Port:      8080,
				Domain:    []string{"localhost"},
				Root:      "./data/sites/default",
				IndexFiles: []string{"index.html", "index.htm"},
				AutoIndex:  true,
				CreatedAt:  time.Now().Format(time.RFC3339),
				UpdatedAt:  time.Now().Format(time.RFC3339),
			},
		},
	}
}

// Load 加载配置文件
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			if err := cfg.Save(path); err != nil {
				return nil, err
			}
			return cfg, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// 验证并修复配置
	cfg.validateAndFix()

	// 检查是否需要迁移
	if cfg.needsMigration() {
		cfg = *migrateConfig(&cfg)
		// 保存迁移后的配置
		_ = cfg.Save(path)
	}

	return &cfg, nil
}

// validateAndFix 验证并修复配置
func (c *Config) validateAndFix() {
	// Legacy HTTP 端口验证
	if c.HTTP.Port <= 0 || c.HTTP.Port > 65535 {
		c.HTTP.Port = 8080
	}

	// FTP 端口验证
	if c.FTP.Port <= 0 || c.FTP.Port > 65535 {
		c.FTP.Port = 2121
	}

	// Admin 端口验证
	if c.Global.AdminPort <= 0 || c.Global.AdminPort > 65535 {
		c.Global.AdminPort = 9527
	}

	// 验证站点配置
	for i := range c.Sites {
		if c.Sites[i].Port < 0 || c.Sites[i].Port > 65535 {
			c.Sites[i].Port = 0
		}
		if c.Sites[i].Type == "" {
			c.Sites[i].Type = "static"
		}
		if len(c.Sites[i].IndexFiles) == 0 {
			c.Sites[i].IndexFiles = []string{"index.html", "index.htm"}
		}
	}
}

// needsMigration 检查是否需要迁移
func (c *Config) needsMigration() bool {
	// 如果没有 Sites 但有 HTTP，需要迁移
	return len(c.Sites) == 0 && c.HTTP.Port > 0
}

// migrateConfig 迁移旧配置到新格式
func migrateConfig(old *Config) *Config {
	newConfig := &Config{
		FTP:   old.FTP,
		Admin: old.Admin,
		Global: GlobalConfig{
			AdminPort: 9527,
			DataDir:   "./data",
		},
		Sites: []SiteConfig{
			{
				ID:        "default",
				Name:      "默认网站",
				Enabled:   true,
				Type:      "static",
				Port:      old.HTTP.Port,
				Root:      old.HTTP.Root,
				IndexFiles: []string{"index.html", "index.htm"},
				AutoIndex:  true,
				CreatedAt:  time.Now().Format(time.RFC3339),
				UpdatedAt:  time.Now().Format(time.RFC3339),
			},
		},
	}

	// 迁移域名
	if old.HTTP.Domain != "" {
		newConfig.Sites[0].Domain = []string{old.HTTP.Domain}
	} else {
		newConfig.Sites[0].Domain = []string{"localhost"}
	}

	return newConfig
}

// Save 保存配置文件
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// CreateDefaultDirectories 创建默认目录
func (c *Config) CreateDefaultDirectories() error {
	dirs := []string{
		c.FTP.Root,
		"./logs",
		c.Global.DataDir,
		filepath.Join(c.Global.DataDir, "ssl"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// 为每个静态站点创建根目录
	for _, site := range c.Sites {
		if site.Type == "static" && site.Root != "" {
			if err := os.MkdirAll(site.Root, 0755); err != nil {
				return err
			}

			// 创建默认 index.html
			indexPath := filepath.Join(site.Root, "index.html")
			if _, err := os.Stat(indexPath); os.IsNotExist(err) {
				defaultIndex := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>` + site.Name + `</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
        .container { text-align: center; background: white; padding: 60px; border-radius: 20px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
        h1 { color: #333; margin-bottom: 10px; }
        p { color: #666; margin-bottom: 30px; }
        .feather { font-size: 80px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="feather">🪶</div>
        <h1>` + site.Name + `</h1>
        <p>小而强悍，无所不能</p>
    </div>
</body>
</html>`
				if err := os.WriteFile(indexPath, []byte(defaultIndex), 0644); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// GetSiteByID 根据 ID 获取站点
func (c *Config) GetSiteByID(id string) *SiteConfig {
	for i := range c.Sites {
		if c.Sites[i].ID == id {
			return &c.Sites[i]
		}
	}
	return nil
}

// AddSite 添加站点
func (c *Config) AddSite(site SiteConfig) error {
	// 检查 ID 是否已存在
	for _, s := range c.Sites {
		if s.ID == site.ID {
			return os.ErrExist
		}
	}

	site.CreatedAt = time.Now().Format(time.RFC3339)
	site.UpdatedAt = time.Now().Format(time.RFC3339)
	c.Sites = append(c.Sites, site)
	return nil
}

// UpdateSite 更新站点
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

// DeleteSite 删除站点
func (c *Config) DeleteSite(id string) bool {
	for i, site := range c.Sites {
		if site.ID == id {
			c.Sites = append(c.Sites[:i], c.Sites[i+1:]...)
			return true
		}
	}
	return false
}
