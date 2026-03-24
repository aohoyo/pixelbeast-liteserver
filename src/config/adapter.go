package config

import (
	_ "time" // 保持兼容
)

// Adapter 配置适配器
// 提供旧配置接口，平滑迁移到新配置结构
type Adapter struct {
	cm *ConfigManager
}

// NewAdapter 创建适配器
func NewAdapter(cm *ConfigManager) *Adapter {
	return &Adapter{cm: cm}
}

// GetSites 获取站点列表
func (a *Adapter) GetSites() []SiteConfig {
	return a.cm.Sites.Sites
}

// GetSiteByID 根据 ID 获取站点
func (a *Adapter) GetSiteByID(id string) *SiteConfig {
	return a.cm.GetSiteByID(id)
}

// GetFTP 获取 FTP 配置
func (a *Adapter) GetFTP() *FTPConfig {
	return a.cm.FTP
}

// GetHTTPPort 获取 HTTP 端口
func (a *Adapter) GetHTTPPort() int {
	if len(a.cm.Sites.Sites) > 0 {
		return a.cm.Sites.Sites[0].Port
	}
	return a.cm.Server.HTTPPort
}

// GetHTTPRoot 获取 HTTP 根目录
func (a *Adapter) GetHTTPRoot() string {
	if len(a.cm.Sites.Sites) > 0 {
		return a.cm.Sites.Sites[0].Root
	}
	return "./web"
}

// GetAdminPort 获取管理端口
func (a *Adapter) GetAdminPort() int {
	return a.cm.Server.AdminPort
}

// GetAdminPath 获取管理入口路径
func (a *Adapter) GetAdminPath() string {
	path := a.cm.Server.AdminPath
	if path == "" {
		path = "/admin"
	}
	return path
}

// GetLogLevel 获取日志级别
func (a *Adapter) GetLogLevel() string {
	return a.cm.Server.Log.Level
}

// Save 保存配置
func (a *Adapter) Save() error {
	return a.cm.Save()
}

// GetConfigManager 获取底层配置管理器
func (a *Adapter) GetConfigManager() *ConfigManager {
	return a.cm
}

// ValidateAdmin 验证管理员密码
func (a *Adapter) ValidateAdmin(username, password string) bool {
	return a.cm.ValidateAdmin(username, password)
}

// ValidateFTPUser 验证 FTP 用户密码
func (a *Adapter) ValidateFTPUser(username, password string) bool {
	return a.cm.ValidateFTPUser(username, password)
}

// GetFTPPort 获取 FTP 端口
func (a *Adapter) GetFTPPort() int {
	return a.cm.FTP.Port
}

// IsFTPEnabled FTP 是否启用
func (a *Adapter) IsFTPEnabled() bool {
	return a.cm.FTP.Enabled
}

// GetFTPRoot 获取 FTP 根目录
func (a *Adapter) GetFTPRoot() string {
	return a.cm.FTP.Root
}

// GetFTPUsers 获取 FTP 用户列表
func (a *Adapter) GetFTPUsers() []FTPUser {
	return a.cm.FTP.Users
}