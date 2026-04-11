package handlers

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"pixelbeast/src/config"
)

// ServerManager 服务管理器
type ServerManager struct {
	mu sync.RWMutex

	// 配置管理器
	ConfigManager *config.ConfigManager

	// 管理面板服务器（独立端口）
	AdminServer  *http.Server
	AdminHandler http.Handler
	AdminPort    int
	adminRunning bool

	// 网站服务器（多站点）
	SitesServer      *http.Server
	SitesRouter      *VirtualHostRouter
	SitesHandler     http.Handler
	sitesRunning     bool
	sitesTLSCfg      *tls.Config
	portServers      map[int]*http.Server // 独立端口站点监听
	portSitesRunning bool

	// HTTP 重定向服务器（端口 80，ACME challenge + HTTPS 重定向）
	redirectServer  *http.Server
	redirectRunning bool

	// FTP 服务
	FTPServer  *FTPServer
	FTPConfig  *config.FTPConfig
	FTPRunning bool

	// SSL 管理
	SSLManager *SSLManager

	// 文件管理
	FileManager *FileManager
}

// NewServerManager 创建服务管理器
func NewServerManager(cm *config.ConfigManager, configPath string) *ServerManager {
	sm := &ServerManager{
		ConfigManager: cm,
		AdminPort:     cm.Server.Admin.Port,
		FileManager:   NewFileManager(),
		SitesRouter:   NewVirtualHostRouter(cm.GetSitesDir()),
		SSLManager:    NewSSLManager("./ssl"),
		portServers:   make(map[int]*http.Server),
	}

	sm.SitesRouter.SetSharedPort(cm.GetSharedPort())

	// 初始化文件管理器书签
	sm.FileManager.UpdateBookmarksFromConfig(cm.Sites.Sites, cm.GetSitesDir())

	return sm
}

// ==================== 管理面板服务器 ====================

// SetAdminHandler 设置管理面板处理器
func (m *ServerManager) SetAdminHandler(handler http.Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AdminHandler = handler
}

// StartAdminPanel 启动管理面板服务器
func (m *ServerManager) StartAdminPanel() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.adminRunning {
		return nil
	}

	if m.AdminHandler == nil {
		return fmt.Errorf("admin handler not set")
	}

	port := m.AdminPort
	if port <= 0 || port > 65535 {
		port = 9527
	}

	// 管理面板直接使用 admin handler（不使用路径前缀）
	// 这样可以直接访问 http://host:port/ 而不需要 /admin 前缀
	m.AdminServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      m.AdminHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	m.adminRunning = true

	go func() {
		LogPanelRuntime(LogLevelInfo, "[Admin] 管理面板启动在端口 %d", port)
		if err := m.AdminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Admin] 服务错误: %v", err)
		}
		m.mu.Lock()
		m.adminRunning = false
		m.mu.Unlock()
	}()

	return nil
}

// StopAdminPanel 停止管理面板服务器
func (m *ServerManager) StopAdminPanel() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.adminRunning || m.AdminServer == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := m.AdminServer.Shutdown(ctx)
	m.adminRunning = false

	LogPanelRuntime(LogLevelInfo, "[Admin] 管理面板已停止")
	return err
}

// RestartAdminPanel 重启管理面板服务器
func (m *ServerManager) RestartAdminPanel() error {
	m.mu.Lock()

	handler := m.AdminHandler
	port := m.AdminPort

	// 停止旧服务
	if m.AdminServer != nil && m.adminRunning {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		done := make(chan error, 1)
		go func() {
			done <- m.AdminServer.Shutdown(ctx)
		}()

		select {
		case err := <-done:
			if err != nil {
				log.Printf("[Admin] 关闭警告: %v", err)
			}
		case <-ctx.Done():
			log.Printf("[Admin] 关闭超时")
		}
		cancel()
	}

	m.adminRunning = false
	m.mu.Unlock()

	time.Sleep(500 * time.Millisecond)

	// 启动新服务
	m.mu.Lock()
	m.AdminServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	m.adminRunning = true

	go func() {
		LogPanelRuntime(LogLevelInfo, "[Admin] 管理面板重启在端口 %d", port)
		if err := m.AdminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Admin] 服务错误: %v", err)
		}
		m.mu.Lock()
		m.adminRunning = false
		m.mu.Unlock()
	}()

	m.mu.Unlock()

	LogPanelRuntime(LogLevelInfo, "[Admin] 管理面板重启完成")
	return nil
}

// IsAdminRunning 检查管理面板是否运行
func (m *ServerManager) IsAdminRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.adminRunning
}

// ==================== 网站服务器 ====================

// SetSitesHandler 设置网站处理器
func (m *ServerManager) SetSitesHandler(handler http.Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SitesHandler = handler
}

// ReloadSites 重新加载站点配置（无需重启）
func (m *ServerManager) ReloadSites() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 使用 ConfigManager 重新加载
	if m.ConfigManager != nil {
		// 重新加载配置文件
		newCM, err := config.NewConfigManager(m.ConfigManager.ConfigDir())
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		m.ConfigManager = newCM
		m.ConfigManager.Sites = newCM.Sites
	}

	// 重建虚拟主机路由
	m.SitesRouter = NewVirtualHostRouter(m.ConfigManager.GetSitesDir())
	m.SitesRouter.SetSharedPort(m.ConfigManager.GetSharedPort())

	for i := range m.ConfigManager.Sites.Sites {
		site := &(m.ConfigManager.Sites.Sites)[i]
		if site.Enabled {
			if err := m.SitesRouter.AddHost(site); err != nil {
				log.Printf("[Sites] 添加站点失败: %s, %v", site.Name, err)
			}
		}
	}

	// 更新文件管理器书签
	m.FileManager.UpdateBookmarksFromConfig(m.ConfigManager.Sites.Sites, m.ConfigManager.GetSitesDir())

	// 重启独立端口站点
	m.startPortSites()

	LogPanelRuntime(LogLevelInfo, "[Sites] 站点配置已重新加载")
	return nil
}

// StartSitesServer 启动网站服务器
func (m *ServerManager) StartSitesServer() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sitesRunning {
		return nil
	}

	// 检查是否有启用的站点
	hasEnabled := false
	for i := range m.ConfigManager.Sites.Sites {
		site := &(m.ConfigManager.Sites.Sites)[i]
		if site.Enabled {
			hasEnabled = true
			if err := m.SitesRouter.AddHost(site); err != nil {
				log.Printf("[Sites] 添加站点失败: %s, %v", site.Name, err)
			}
		}
	}

	if !hasEnabled {
		LogPanelRuntime(LogLevelInfo, "[Sites] 无启用的站点，跳过网站服务器启动")
		return nil
	}

	// 每个站点独立端口监听
	m.startPortSites()


	// 启动 HTTP 重定向（端口 80，ACME challenge + HTTPS 重定向）
	m.StartHTTPRedirect()
	m.sitesRunning = true
	LogPanelRuntime(LogLevelInfo, "[Sites] 站点服务已启动")
	return nil
}

// startPortSites 启动独立端口站点
func (m *ServerManager) startPortSites() {
	m.stopPortSites()

	for i := range m.ConfigManager.Sites.Sites {
		site := &(m.ConfigManager.Sites.Sites)[i]
		if !site.Enabled {
			continue
		}
		if site.Port <= 0 {
			continue
		}
		// 跳过管理面板端口
		if site.Port == m.ConfigManager.Server.Admin.Port {
			continue
		}

		handler, err := m.SitesRouter.createHandler(site)
		if err != nil {
			log.Printf("[Sites] 独立端口 %d 创建处理器失败: %v", site.Port, err)
			continue
		}

		srv := &http.Server{
			Addr:         fmt.Sprintf(":%d", site.Port),
			Handler:      handler,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		}

		// SSL/TLS 支持
		sslEnabled := site.SSL != nil && site.SSL.Enabled
		if sslEnabled && m.SSLManager != nil {
			srv.TLSConfig = &tls.Config{
				GetCertificate: m.SSLManager.GetCertificate,
			}
		}
		m.portServers[site.Port] = srv

		go func(port int, name string, useTLS bool) {
			if useTLS {
				LogPanelRuntime(LogLevelInfo, "[Sites] 站点 %s 启动TLS在独立端口 %d", name, port)
				if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
					log.Printf("[Sites] 端口 %d TLS服务错误: %v", port, err)
				}
			} else {
				LogPanelRuntime(LogLevelInfo, "[Sites] 站点 %s 启动在独立端口 %d", name, port)
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("[Sites] 端口 %d 服务错误: %v", port, err)
				}
			}
		}(site.Port, site.Name, sslEnabled)
	}
	m.portSitesRunning = true
}

// stopPortSites 停止独立端口站点
func (m *ServerManager) stopPortSites() {
	for port, srv := range m.portServers {
		srv.Close()
		LogPanelRuntime(LogLevelInfo, "[Sites] 独立端口 %d 已停止", port)
	}
	m.portServers = make(map[int]*http.Server)
	m.portSitesRunning = false
}

// StopSitesServer 停止网站服务器
func (m *ServerManager) StopSitesServer() error {
	m.mu.Lock()
	if !m.sitesRunning {
		m.mu.Unlock()
		return nil
	}
	m.sitesRunning = false
	m.mu.Unlock()

	m.stopPortSites()
	m.stopHTTPRedirect()

	LogPanelRuntime(LogLevelInfo, "[Sites] 网站服务器已停止")
	return nil
}

// RestartSitesServer 重启网站服务器
func (m *ServerManager) RestartSitesServer() error {
	if err := m.StopSitesServer(); err != nil {
		return err
	}
	time.Sleep(500 * time.Millisecond)
	return m.StartSitesServer()
}

// IsSitesRunning 检查网站服务器是否运行
func (m *ServerManager) IsSitesRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sitesRunning
}

// StartSite 启动单个站点
func (m *ServerManager) StartSite(siteID string) error {
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
		m.ConfigManager.Save()
	}

	m.startSitePort(site)
	LogPanelRuntime(LogLevelInfo, "[Sites] 站点 %s 已启动", site.Name)
	return nil
}

// StopSite 停止单个站点
func (m *ServerManager) StopSite(siteID string) error {
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
		m.ConfigManager.Save()
	}

	LogPanelRuntime(LogLevelInfo, "[Sites] 站点 %s 已停止", site.Name)
	return nil
}

// AddSiteRuntime 添加站点并立即启动（无需全量重载）
func (m *ServerManager) AddSiteRuntime(site *config.SiteConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !site.Enabled {
		return
	}

	if err := m.SitesRouter.AddHost(site); err != nil {
		log.Printf("[Sites] 添加站点路由失败: %s, %v", site.Name, err)
		return
	}

	m.startSitePort(site)
	LogPanelRuntime(LogLevelInfo, "[Sites] 站点 %s 已添加并启动", site.Name)
}

// UpdateSiteRuntime 更新站点（先停旧再启新，无需全量重载）
func (m *ServerManager) UpdateSiteRuntime(site *config.SiteConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SitesRouter.RemoveHost(site.ID)
	m.stopSitePort(site)

	if !site.Enabled {
		LogPanelRuntime(LogLevelInfo, "[Sites] 站点 %s 已更新（停止状态）", site.Name)
		return
	}

	if err := m.SitesRouter.AddHost(site); err != nil {
		log.Printf("[Sites] 更新站点路由失败: %s, %v", site.Name, err)
		return
	}

	m.startSitePort(site)
	LogPanelRuntime(LogLevelInfo, "[Sites] 站点 %s 已更新并重启", site.Name)
}

// DeleteSiteRuntime 删除站点并立即停止（无需全量重载）
func (m *ServerManager) DeleteSiteRuntime(siteID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SitesRouter.RemoveHost(siteID)

	if site := m.ConfigManager.GetSiteByID(siteID); site != nil {
		m.stopSitePort(site)
	}

	LogPanelRuntime(LogLevelInfo, "[Sites] 站点 %s 已删除", siteID)
}

// startSitePort 启动单个站点的端口监听
func (m *ServerManager) startSitePort(site *config.SiteConfig) {
	if site.Port <= 0 || site.Port == m.ConfigManager.Server.Admin.Port {
		return
	}

	handler, err := m.SitesRouter.createHandler(site)
	if err != nil {
		log.Printf("[Sites] 端口 %d 创建处理器失败: %v", site.Port, err)
		return
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", site.Port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// SSL/TLS 支持
	sslEnabled := site.SSL != nil && site.SSL.Enabled
	if sslEnabled && m.SSLManager != nil {
		srv.TLSConfig = &tls.Config{
			GetCertificate: m.SSLManager.GetCertificate,
		}
	}
	m.portServers[site.Port] = srv

	go func(port int, name string, useTLS bool) {
		if useTLS {
			LogPanelRuntime(LogLevelInfo, "[Sites] 站点 %s 启动TLS在端口 %d", name, port)
			if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				log.Printf("[Sites] 端口 %d TLS服务错误: %v", port, err)
			}
		} else {
			LogPanelRuntime(LogLevelInfo, "[Sites] 站点 %s 启动在端口 %d", name, port)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("[Sites] 端口 %d 服务错误: %v", port, err)
			}
		}
	}(site.Port, site.Name, sslEnabled)
}

// stopSitePort 停止单个站点的端口监听
func (m *ServerManager) stopSitePort(site *config.SiteConfig) {
	if site.Port <= 0 {
		return
	}
	if srv, ok := m.portServers[site.Port]; ok {
		delete(m.portServers, site.Port)
		srv.Close()
		LogPanelRuntime(LogLevelInfo, "[Sites] 端口 %d 已关闭", site.Port)
	}
}

// RestartSite 重启单个站点
func (m *ServerManager) RestartSite(siteID string) error {
	// 停止
	if err := m.StopSite(siteID); err != nil {
		return err
	}

	time.Sleep(300 * time.Millisecond)

	// 启动
	return m.StartSite(siteID)
}

// ==================== FTP 服务（保持不变）====================

// SetFTPServer 设置 FTP 服务器
func (m *ServerManager) SetFTPServer(server *FTPServer, cfg *config.FTPConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FTPServer = server
	m.FTPConfig = cfg
}

// SyncFTPConfig 同步 FTP 服务器配置指针和验证器
// 当 ConfigManager 重新加载后，FTP 服务器的 Config 指针和 validator 需要更新
func (m *ServerManager) SyncFTPConfig() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.FTPServer != nil && m.ConfigManager != nil {
		m.FTPServer.Config = m.ConfigManager.FTP
		m.FTPConfig = m.ConfigManager.FTP
		m.FTPServer.SetValidator(m.ConfigManager)
	}
}

// StartFTP 启动 FTP 服务
func (m *ServerManager) StartFTP() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.FTPRunning {
		return nil
	}

	if m.FTPServer == nil {
		return fmt.Errorf("FTP 服务器未初始化")
	}

	if err := m.FTPServer.Start(); err != nil {
		return err
	}

	m.FTPRunning = true

	LogPanelRuntime(LogLevelInfo, "[FTP] 服务已启动")
	return nil
}

// StopFTP 停止 FTP 服务
func (m *ServerManager) StopFTP() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.FTPRunning || m.FTPServer == nil {
		return nil
	}

	if err := m.FTPServer.Stop(); err != nil {
		return err
	}

	m.FTPRunning = false

	LogPanelRuntime(LogLevelInfo, "[FTP] 服务已停止")
	return nil
}

// RestartFTP 重启 FTP 服务
func (m *ServerManager) RestartFTP() error {
	m.mu.Lock()

	if m.FTPServer != nil && m.FTPRunning {
		m.FTPServer.Stop()
		m.FTPRunning = false
	}

	m.mu.Unlock()

	time.Sleep(500 * time.Millisecond)

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.FTPServer == nil {
		return fmt.Errorf("FTP 服务器未初始化")
	}

	if err := m.FTPServer.Start(); err != nil {
		return err
	}

	m.FTPRunning = true
	LogPanelRuntime(LogLevelInfo, "[FTP] 服务重启完成")

	return nil
}

// IsFTPRunning 检查 FTP 服务是否运行
func (m *ServerManager) IsFTPRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.FTPRunning
}

// ==================== HTTP 重定向 ====================

// startHTTPRedirect 启动端口 80 的 HTTP 重定向服务器
// 用于 ACME challenge 和 HTTP→HTTPS 重定向
func (m *ServerManager) StartHTTPRedirect() {
	if m.SSLManager == nil {
		return
	}
	if !m.SSLManager.HasSSLDomains() && !m.SSLManager.HasPendingChallenges() {
		return
	}
	// 检查是否已有运行中的重定向服务
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
		LogPanelRuntime(LogLevelInfo, "[SSL] HTTP 重定向服务启动在端口 80")
		if err := m.redirectServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[SSL] 端口 80 服务错误: %v", err)
		}
		m.mu.Lock()
		m.redirectRunning = false
		m.mu.Unlock()
	}()
}

// stopHTTPRedirect 停止 HTTP 重定向服务器
func (m *ServerManager) stopHTTPRedirect() {
	if m.redirectServer != nil {
		m.redirectServer.Close()
		m.redirectServer = nil
		m.redirectRunning = false
		LogPanelRuntime(LogLevelInfo, "[SSL] HTTP 重定向服务已停止")
	}
}

// ==================== 配置 ====================

// ReloadConfig 重新加载配置
func (m *ServerManager) ReloadConfig() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 使用 ConfigManager 重新加载
	if m.ConfigManager != nil {
		newCM, err := config.NewConfigManager(m.ConfigManager.ConfigDir())
		if err != nil {
			return err
		}
		m.ConfigManager = newCM
	}

	return nil
}

// ==================== 状态信息 ====================

// GetStatus 获取所有服务状态
func (m *ServerManager) GetStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 获取站点信息
	sites := make([]map[string]interface{}, 0)
	for _, site := range m.ConfigManager.Sites.Sites {
		sites = append(sites, map[string]interface{}{
			"id":      site.ID,
			"name":    site.Name,
			"enabled": site.Enabled,
			"type":    site.Type,
			"port":    site.Port,
			"domains": site.Domain,
		})
	}

	return map[string]interface{}{
		"admin": map[string]interface{}{
			"running": m.adminRunning,
			"port":    m.AdminPort,
		},
		"sites": map[string]interface{}{
			"running": m.sitesRunning,
			"count":   len(m.ConfigManager.Sites.Sites),
			"items":   sites,
		},
		"ftp": map[string]interface{}{
			"running": m.FTPRunning,
			"port":    m.FTPConfig.Port,
		},
	}
}
