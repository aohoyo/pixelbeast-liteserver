package site

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"pixelbeast/src/config"
	"pixelbeast/src/logger"
	"pixelbeast/src/file"
	"pixelbeast/src/ssl"
)

// SiteManager 站点管理器（只管站点启停、虚拟主机）
type SiteManager struct {
	mu sync.RWMutex

	// 配置管理器
	ConfigManager *config.ConfigManager

	// 网站服务器（多站点）
	SitesRouter      *VirtualHostRouter
	sitesRunning     bool
	portServers      map[int]*http.Server // 独立端口站点监听

	// HTTP 重定向服务器（端口 80，ACME challenge + HTTPS 重定向）
	redirectServer  *http.Server
	redirectRunning bool

	// SSL 管理
	SSLManager *ssl.SSLManager

	// 文件管理
	FileManager *file.FileManager
}

// NewSiteManager 创建站点管理器
func NewSiteManager(cm *config.ConfigManager, sslMgr *ssl.SSLManager, fileMgr *file.FileManager) *SiteManager {
	sm := &SiteManager{
		ConfigManager: cm,
		FileManager:   fileMgr,
		SitesRouter:   NewVirtualHostRouter(cm.GetSitesDir()),
		SSLManager:    sslMgr,
		portServers:   make(map[int]*http.Server),
	}

	sm.SitesRouter.SetSharedPort(cm.GetSharedPort())
	sm.FileManager.UpdateBookmarksFromConfig(cm.Sites.Sites, cm.GetSitesDir())

	return sm
}

// ==================== 网站服务器 ====================

// ReloadSites 重新加载站点配置（无需重启）
func (m *SiteManager) ReloadSites() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ConfigManager != nil {
		newCM, err := config.NewConfigManager(m.ConfigManager.ConfigDir())
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		m.ConfigManager = newCM
		m.ConfigManager.Sites = newCM.Sites
	}

	m.SitesRouter = NewVirtualHostRouter(m.ConfigManager.GetSitesDir())
	m.SitesRouter.SetSharedPort(m.ConfigManager.GetSharedPort())

	for i := range m.ConfigManager.Sites.Sites {
		site := &(m.ConfigManager.Sites.Sites)[i]
		if site.Enabled {
			if err := m.SitesRouter.AddHost(site); err != nil {
				logger.LogPanelRuntime(logger.LogLevelError, "[Sites] 添加站点失败: %s, %v", site.Name, err)
			}
		}
	}

	m.FileManager.UpdateBookmarksFromConfig(m.ConfigManager.Sites.Sites, m.ConfigManager.GetSitesDir())
	m.startPortSites()

	logger.LogPanelRuntime(logger.LogLevelInfo, "[Sites] 站点配置已重新加载")
	return nil
}

// StartSitesServer 启动网站服务器
func (m *SiteManager) StartSitesServer() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sitesRunning {
		return nil
	}

	hasEnabled := false
	for i := range m.ConfigManager.Sites.Sites {
		site := &(m.ConfigManager.Sites.Sites)[i]
		if site.Enabled {
			hasEnabled = true
			if err := m.SitesRouter.AddHost(site); err != nil {
				logger.LogPanelRuntime(logger.LogLevelError, "[Sites] 添加站点失败: %s, %v", site.Name, err)
			}
		}
	}

	if !hasEnabled {
		logger.LogPanelRuntime(logger.LogLevelInfo, "[Sites] 无启用的站点，跳过网站服务器启动")
		return nil
	}

	m.startPortSites()
	m.StartHTTPRedirect()
	m.sitesRunning = true
	logger.LogPanelRuntime(logger.LogLevelInfo, "[Sites] 站点服务已启动")
	return nil
}

func (m *SiteManager) startPortSites() {
	m.stopPortSites()

	for i := range m.ConfigManager.Sites.Sites {
		site := &(m.ConfigManager.Sites.Sites)[i]
		if !site.Enabled {
			continue
		}
		m.startSitePort(site)
	}
}

func (m *SiteManager) stopPortSites() {
	for port, srv := range m.portServers {
		srv.Close()
		logger.LogPanelRuntime(logger.LogLevelInfo, "[Sites] 独立端口 %d 已停止", port)
	}
	m.portServers = make(map[int]*http.Server)
}

// StopSitesServer 停止网站服务器
func (m *SiteManager) StopSitesServer() error {
	m.mu.Lock()
	if !m.sitesRunning {
		m.mu.Unlock()
		return nil
	}
	m.sitesRunning = false
	m.mu.Unlock()

	m.stopPortSites()
	m.stopHTTPRedirect()

	logger.LogPanelRuntime(logger.LogLevelInfo, "[Sites] 网站服务器已停止")
	return nil
}

// RestartSitesServer 重启网站服务器
func (m *SiteManager) RestartSitesServer() error {
	if err := m.StopSitesServer(); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)
	return m.StartSitesServer()
}

// IsSitesRunning 检查网站服务器是否运行
func (m *SiteManager) IsSitesRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sitesRunning
}

// StartSite 启动单个站点
func (m *SiteManager) StartSite(siteID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	site := m.ConfigManager.GetSiteByID(siteID)
	if site == nil {
		return fmt.Errorf("站点不存在")
	}

	if site.Enabled {
		return nil
	}

	site.Enabled = true
	site.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := m.SitesRouter.AddHost(site); err != nil {
		site.Enabled = false
		return fmt.Errorf("添加站点路由失败: %w", err)
	}

	if m.ConfigManager != nil {
		if err := m.ConfigManager.Save(); err != nil {
			logger.LogPanelRuntime(logger.LogLevelError, "[Sites] 保存配置失败: %v", err)
		}
	}

	m.startSitePort(site)
	logger.LogPanelRuntime(logger.LogLevelInfo, "[Sites] 站点 %s 已启动", site.Name)
	return nil
}

// StopSite 停止单个站点
func (m *SiteManager) StopSite(siteID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	site := m.ConfigManager.GetSiteByID(siteID)
	if site == nil {
		return fmt.Errorf("站点不存在")
	}

	if !site.Enabled {
		return nil
	}

	m.SitesRouter.RemoveHost(siteID)
	m.stopSitePort(site)

	site.Enabled = false
	site.UpdatedAt = time.Now().Format(time.RFC3339)

	if m.ConfigManager != nil {
		if err := m.ConfigManager.Save(); err != nil {
			logger.LogPanelRuntime(logger.LogLevelError, "[Sites] 保存配置失败: %v", err)
		}
	}

	logger.LogPanelRuntime(logger.LogLevelInfo, "[Sites] 站点 %s 已停止", site.Name)
	return nil
}

// AddSiteRuntime 添加站点并立即启动（无需全量重载）
func (m *SiteManager) AddSiteRuntime(site *config.SiteConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !site.Enabled {
		return
	}

	if err := m.SitesRouter.AddHost(site); err != nil {
		logger.LogPanelRuntime(logger.LogLevelError, "[Sites] 添加站点路由失败: %s, %v", site.Name, err)
		return
	}

	m.startSitePort(site)
	logger.LogPanelRuntime(logger.LogLevelInfo, "[Sites] 站点 %s 已添加并启动", site.Name)
}

// UpdateSiteRuntime 更新站点（先停旧再启新，无需全量重载）
func (m *SiteManager) UpdateSiteRuntime(site *config.SiteConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SitesRouter.RemoveHost(site.ID)
	m.stopSitePort(site)

	if !site.Enabled {
		logger.LogPanelRuntime(logger.LogLevelInfo, "[Sites] 站点 %s 已更新（停止状态）", site.Name)
		return
	}

	if err := m.SitesRouter.AddHost(site); err != nil {
		logger.LogPanelRuntime(logger.LogLevelError, "[Sites] 更新站点路由失败: %s, %v", site.Name, err)
		return
	}

	m.startSitePort(site)
	logger.LogPanelRuntime(logger.LogLevelInfo, "[Sites] 站点 %s 已更新并重启", site.Name)
}

// DeleteSiteRuntime 删除站点并立即停止（无需全量重载）
func (m *SiteManager) DeleteSiteRuntime(siteID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SitesRouter.RemoveHost(siteID)

	if site := m.ConfigManager.GetSiteByID(siteID); site != nil {
		m.stopSitePort(site)
	}

	logger.LogPanelRuntime(logger.LogLevelInfo, "[Sites] 站点 %s 已删除", siteID)
}

func (m *SiteManager) hstsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

func (m *SiteManager) startSitePort(site *config.SiteConfig) {
	if site.Port <= 0 || site.Port == m.ConfigManager.Server.Admin.Port {
		return
	}

	handler, err := m.SitesRouter.createHandler(site)
	if err != nil {
		logger.LogPanelRuntime(logger.LogLevelError, "[Sites] 端口 %d 创建处理器失败: %v", site.Port, err)
		return
	}

	sslEnabled := site.SSL != nil && site.SSL.Enabled
	if sslEnabled && m.SSLManager != nil {
		if site.SSL.HSTS {
			handler = m.hstsMiddleware(handler)
		}
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", site.Port),
		Handler:           handler,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if sslEnabled && m.SSLManager != nil {
		srv.TLSConfig = &tls.Config{
			GetCertificate: m.SSLManager.GetCertificate,
		}
	}
	m.portServers[site.Port] = srv

	go func(port int, name string, useTLS bool) {
		if useTLS {
			logger.LogPanelRuntime(logger.LogLevelInfo, "[Sites] 站点 %s 启动TLS在端口 %d", name, port)
			if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				logger.LogPanelRuntime(logger.LogLevelError, "[Sites] 端口 %d TLS服务错误: %v", port, err)
			}
		} else {
			logger.LogPanelRuntime(logger.LogLevelInfo, "[Sites] 站点 %s 启动在端口 %d", name, port)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.LogPanelRuntime(logger.LogLevelError, "[Sites] 端口 %d 服务错误: %v", port, err)
			}
		}
	}(site.Port, site.Name, sslEnabled)
}

func (m *SiteManager) stopSitePort(site *config.SiteConfig) {
	if site.Port <= 0 {
		return
	}
	if srv, ok := m.portServers[site.Port]; ok {
		delete(m.portServers, site.Port)
		srv.Close()
		logger.LogPanelRuntime(logger.LogLevelInfo, "[Sites] 端口 %d 已关闭", site.Port)
	}
}

// RestartSite 重启单个站点
func (m *SiteManager) RestartSite(siteID string) error {
	if err := m.StopSite(siteID); err != nil {
		return err
	}
	time.Sleep(300 * time.Millisecond)
	return m.StartSite(siteID)
}

// ==================== HTTP 重定向 ====================

// StartHTTPRedirect 启动端口 80 的 HTTP 重定向服务器
func (m *SiteManager) StartHTTPRedirect() {
	if m.SSLManager == nil {
		return
	}
	if !m.SSLManager.HasSSLDomains() && !m.SSLManager.HasPendingChallenges() {
		return
	}
	if m.redirectRunning && m.redirectServer != nil {
		return
	}

	handler := m.SSLManager.GetHTTPSRedirectHandler(nil)
	m.redirectServer = &http.Server{
		Addr:    ":80",
		Handler: handler,
	}
	m.redirectRunning = true

	go func() {
		logger.LogPanelRuntime(logger.LogLevelInfo, "[SSL] HTTP 重定向服务启动在端口 80")
		if err := m.redirectServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.LogPanelRuntime(logger.LogLevelError, "[SSL] 端口 80 服务错误: %v", err)
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		m.redirectRunning = false
	}()
}

// EnsureHTTPRedirect 确保端口 80 HTTP 重定向服务器运行（用于 SSL 证书验证）
func (m *SiteManager) EnsureHTTPRedirect() {
	if m.SSLManager == nil {
		return
	}
	if m.redirectRunning && m.redirectServer != nil {
		return
	}

	handler := m.SSLManager.GetHTTPSRedirectHandler(nil)
	m.redirectServer = &http.Server{
		Addr:    ":80",
		Handler: handler,
	}
	m.redirectRunning = true

	go func() {
		logger.LogPanelRuntime(logger.LogLevelInfo, "[SSL] HTTP 重定向服务启动在端口 80（证书验证模式）")
		if err := m.redirectServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.LogPanelRuntime(logger.LogLevelError, "[SSL] 端口 80 服务错误: %v", err)
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		m.redirectRunning = false
	}()
}

func (m *SiteManager) stopHTTPRedirect() {
	if m.redirectServer != nil {
		m.redirectServer.Close()
		m.redirectServer = nil
		m.redirectRunning = false
		logger.LogPanelRuntime(logger.LogLevelInfo, "[SSL] HTTP 重定向服务已停止")
	}
}

// ==================== 配置 ====================

// ReloadConfig 重新加载配置
func (m *SiteManager) ReloadConfig() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ConfigManager != nil {
		newCM, err := config.NewConfigManager(m.ConfigManager.ConfigDir())
		if err != nil {
			return err
		}
		m.ConfigManager = newCM
	}

	return nil
}
