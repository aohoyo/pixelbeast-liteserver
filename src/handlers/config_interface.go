package handlers

import "pixelbeast/src/config"

// ConfigInterface 统一配置接口
type ConfigInterface interface {
	// 站点管理
	GetSites() []config.SiteConfig
	GetSiteByID(id string) *config.SiteConfig

	// FTP 配置
	GetFTP() *config.FTPConfig

	// HTTP 配置（用于兼容）
	GetHTTPPort() int
	GetHTTPRoot() string

	// 全局配置
	GetAdminPort() int

	// 日志配置
	GetLogLevel() string

	// 保存
	Save() error
}