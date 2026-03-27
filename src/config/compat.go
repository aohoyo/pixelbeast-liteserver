package config

import (
	"time"
)

// ==================== 旧配置结构（兼容层）====================
// 这些类型用于内存中的配置表示

// Config 旧配置结构
type Config struct {
	Global GlobalConfig
	Admin  AdminConfig
	HTTP   HTTPConfig
	FTP    FTPConfig
	Sites  []SiteConfig
	Log    LogConfig
}

// GlobalConfig 全局配置
type GlobalConfig struct {
	AdminPort int    `json:"admin_port"`
	FTPDir    string `json:"ftp_dir"`
	BackupDir string `json:"backup_dir"`
}

// AdminConfig 管理员配置
type AdminConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Path     string `json:"path"`
}

// HTTPConfig HTTP 配置
type HTTPConfig struct {
	Port int    `json:"port"`
	Root string `json:"root"`
}

// LogConfig 日志配置
type LogConfig struct {
	RetentionDays int               `json:"retention_days"`
	MaxSizeMB     int               `json:"max_size_mb"`
	CompressDays  int               `json:"compress_days"`
	CleanupHour   int               `json:"cleanup_hour"`
	Level         string            `json:"level"`
	Levels        map[string]string `json:"levels,omitempty"` // 各分类级别
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
	for i := range c.Sites {
		if c.Sites[i].ID == id {
			c.Sites = append(c.Sites[:i], c.Sites[i+1:]...)
			return true
		}
	}
	return false
}