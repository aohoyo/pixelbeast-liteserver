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

	// 管理面板服务器（独立端口）
	AdminServer  *http.Server
	AdminHandler http.Handler
	AdminPort    int
	adminRunning bool

	// 网站服务器（多站点）
	SitesServer  *http.Server
	SitesRouter  *VirtualHostRouter
	SitesHandler http.Handler
	sitesRunning bool
	sitesTLSCfg  *tls.Config

	// FTP 服务
	FTPServer  *FTPServer
	FTPConfig  *config.FTPConfig
	FTPRunning bool

	// SSL 管理
	SSLManager *SSLManager

	// 文件管理
	FileManager *FileManager

	// 配置
	Config     *config.Config
	ConfigPath string
}

// NewServerManager 创建服务管理器
func NewServerManager(cfg *config.Config, configPath string) *ServerManager {
	sm := &ServerManager{
		Config:      cfg,
		ConfigPath:  configPath,
		AdminPort:   cfg.Global.AdminPort,
		FileManager: NewFileManager(),
		SitesRouter: NewVirtualHostRouter(),
		SSLManager:  NewSSLManager(getSSLDir(cfg)),
	}

	// 初始化文件管理器书签
	sm.FileManager.UpdateBookmarksFromConfig(cfg.Sites)

	return sm
}

// getSSLDir 获取 SSL 证书目录
func getSSLDir(cfg *config.Config) string {
	return "./ssl" // SSL 证书固定存储在程序运行目录下的 ssl 目录
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
		log.Printf("[Admin] 管理面板启动在端口 %d", port)
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

	log.Printf("[Admin] 管理面板已停止")
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
		log.Printf("[Admin] 管理面板重启在端口 %d", port)
		if err := m.AdminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[Admin] 服务错误: %v", err)
		}
		m.mu.Lock()
		m.adminRunning = false
		m.mu.Unlock()
	}()

	m.mu.Unlock()

	log.Printf("[Admin] 管理面板重启完成")
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

	// 重新加载配置
	cfg, err := config.Load(m.ConfigPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	m.Config = cfg

	// 重建虚拟主机路由
	m.SitesRouter = NewVirtualHostRouter()

	for i := range cfg.Sites {
		site := &cfg.Sites[i]
		if site.Enabled {
			if err := m.SitesRouter.AddHost(site); err != nil {
				log.Printf("[Sites] 添加站点失败: %s, %v", site.Name, err)
			}
		}
	}

	// 更新文件管理器书签
	m.FileManager.UpdateBookmarksFromConfig(cfg.Sites)

	log.Printf("[Sites] 站点配置已重新加载")
	return nil
}

// StartSitesServer 启动网站服务器
func (m *ServerManager) StartSitesServer() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sitesRunning {
		return nil
	}

	// 初始化虚拟主机路由器
	for i := range m.Config.Sites {
		site := &m.Config.Sites[i]
		if site.Enabled {
			if err := m.SitesRouter.AddHost(site); err != nil {
				log.Printf("[Sites] 添加站点失败: %s, %v", site.Name, err)
			}
		}
	}

	// 检查是否有站点需要 HTTPS
	hasHTTPS := false
	for _, site := range m.Config.Sites {
		if site.SSL != nil && site.SSL.Enabled {
			hasHTTPS = true
			break
		}
	}

	// 启动 HTTP 服务器
	if m.SitesHandler == nil {
		m.SitesHandler = m.SitesRouter
	}

	// 获取共享端口（大多数站点使用的端口）
	sharedPort := 8080
	for _, site := range m.Config.Sites {
		if site.Enabled && site.Port > 0 && site.Port != sharedPort {
			// 如果有独立端口的站点，需要特殊处理
			// 暂时简化：只启动共享端口的服务器
		}
	}

	m.SitesServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", sharedPort),
		Handler:      m.SitesRouter,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// 配置 TLS（如果需要）
	if hasHTTPS && m.sitesTLSCfg != nil {
		m.SitesServer.TLSConfig = m.sitesTLSCfg
	}

	m.sitesRunning = true

	go func() {
		log.Printf("[Sites] 网站服务器启动在端口 %d", sharedPort)
		if m.SitesServer.TLSConfig != nil {
			if err := m.SitesServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				log.Printf("[Sites] HTTPS 服务错误: %v", err)
			}
		} else {
			if err := m.SitesServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("[Sites] HTTP 服务错误: %v", err)
			}
		}
		m.mu.Lock()
		m.sitesRunning = false
		m.mu.Unlock()
	}()

	return nil
}

// StopSitesServer 停止网站服务器
func (m *ServerManager) StopSitesServer() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.sitesRunning || m.SitesServer == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := m.SitesServer.Shutdown(ctx)
	m.sitesRunning = false

	log.Printf("[Sites] 网站服务器已停止")
	return err
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

// ==================== FTP 服务（保持不变）====================

// SetFTPServer 设置 FTP 服务器
func (m *ServerManager) SetFTPServer(server *FTPServer, cfg *config.FTPConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FTPServer = server
	m.FTPConfig = cfg
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
	m.Config.FTP.Enabled = true
	m.Config.Save(m.ConfigPath)

	log.Printf("[FTP] 服务已启动")
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
	m.Config.FTP.Enabled = false
	m.Config.Save(m.ConfigPath)

	log.Printf("[FTP] 服务已停止")
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
	log.Printf("[FTP] 服务重启完成")

	return nil
}

// IsFTPRunning 检查 FTP 服务是否运行
func (m *ServerManager) IsFTPRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.FTPRunning
}

// ==================== 配置 ====================

// ReloadConfig 重新加载配置
func (m *ServerManager) ReloadConfig() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg, err := config.Load(m.ConfigPath)
	if err != nil {
		return err
	}

	m.Config = cfg
	return nil
}

// ==================== 状态信息 ====================

// GetStatus 获取所有服务状态
func (m *ServerManager) GetStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 获取站点信息
	sites := make([]map[string]interface{}, 0)
	for _, site := range m.Config.Sites {
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
			"count":   len(m.Config.Sites),
			"items":   sites,
		},
		"ftp": map[string]interface{}{
			"running": m.FTPRunning,
			"port":    m.FTPConfig.Port,
		},
	}
}
