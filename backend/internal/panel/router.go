package panel

import (
	"net/http"
	"sync"
)

// setupRouter 创建路由表
func (h *Handler) setupRouter() http.Handler {
	mux := http.NewServeMux()

	// ==================== 公开路由 ====================

	// 页面
	mux.HandleFunc("/login", h.loginPage)
	mux.HandleFunc("/favicon.ico", h.serveFavicon)

	// 认证 API
	mux.HandleFunc("/api/login", h.loginAPI)
	mux.HandleFunc("/api/logout", h.logoutAPI)

	// 静态资源（原生版）
	mux.HandleFunc("/css/", h.serveCSS)
	mux.HandleFunc("/js/", h.serveJS)
	mux.HandleFunc("/components/", h.serveComponents)
	mux.HandleFunc("/images/", h.serveImages)
	mux.HandleFunc("/icons/", h.serveIcons)

	// 静态资源（Vue 版：Vite 构建产物）
	mux.HandleFunc("/assets/", h.serveVueAssets)

	// ==================== 认证路由 ====================
	authMux := http.NewServeMux()

	// 状态
	authMux.HandleFunc("/api/auth/change-password", h.handleChangePassword)

	// 系统监控
	authMux.HandleFunc("/api/system/status", h.getSystemStatus)
	authMux.HandleFunc("/api/system/free-memory", h.freeMemory)
	authMux.HandleFunc("/api/system/cleanup-scan", h.scanCleanup)
	authMux.HandleFunc("/api/system/cleanup", h.executeCleanup)
	authMux.HandleFunc("/api/system/time/sync", h.syncSystemTime)
	authMux.HandleFunc("/api/system/restart", h.restartServer)
	authMux.HandleFunc("/api/system/check-update", h.checkUpdate)

	// 配置
	authMux.HandleFunc("/api/config", h.getConfig)
	authMux.HandleFunc("/api/config/save", h.saveConfig)
	authMux.HandleFunc("/api/config/reset", h.resetConfig)

	// 备份管理
	authMux.HandleFunc("/api/backups", h.listBackups)
	authMux.HandleFunc("/api/backups/create", h.createBackup)
	authMux.HandleFunc("/api/backups/delete", h.deleteBackup)
	authMux.HandleFunc("/api/backups/download", h.downloadBackup)
	authMux.HandleFunc("/api/backups/restore", h.restoreBackup)

	// HTTP 文件管理
	authMux.HandleFunc("/api/files", h.listFiles)
	authMux.HandleFunc("/api/files/quick-dirs", h.getQuickDirs)
	authMux.HandleFunc("/api/files/upload/chunk", h.uploadChunk)
	authMux.HandleFunc("/api/files/upload/merge", h.mergeChunks)
	authMux.HandleFunc("/api/files/upload/status", h.uploadChunkStatus)
	authMux.HandleFunc("/api/files/upload/path", h.uploadFileWithPath)
	authMux.HandleFunc("/api/files/delete", h.deleteFile)
	authMux.HandleFunc("/api/files/mkdir", h.mkdir)
	authMux.HandleFunc("/api/files/rename", h.renameFile)
	authMux.HandleFunc("/api/files/download", h.downloadFile)
	authMux.HandleFunc("/api/files/copy", h.copyFile)
	authMux.HandleFunc("/api/files/touch", h.touchFile)
	authMux.HandleFunc("/api/files/move", h.moveFile)
	authMux.HandleFunc("/api/files/chmod", h.chmodFile)
	authMux.HandleFunc("/api/files/permissions", h.getFilePermissions)
	authMux.HandleFunc("/api/files/read", h.readFileContent)
	authMux.HandleFunc("/api/files/run", h.handleRunScript)
	authMux.HandleFunc("/api/files/processes", h.handleListProcesses)
	authMux.HandleFunc("/api/files/processes/output", h.handleProcessOutput)
	authMux.HandleFunc("/api/files/processes/stop", h.handleStopProcess)
	authMux.HandleFunc("/api/files/processes/delete", h.handleDeleteProcess)
	authMux.HandleFunc("/api/files/save", h.saveFileContent)
	authMux.HandleFunc("/api/files/compress", h.compressFiles)
	authMux.HandleFunc("/api/files/extract", h.extractFile)
	authMux.HandleFunc("/api/files/share", h.shareFile)
	authMux.HandleFunc("/api/files/share/list", h.listShareLinks)
	authMux.HandleFunc("/api/files/share/delete", h.deleteShareLink)

	// 回收站
	authMux.HandleFunc("/api/files/trash/list", h.listTrash)
	authMux.HandleFunc("/api/files/trash/restore", h.restoreTrash)
	authMux.HandleFunc("/api/files/trash/delete", h.permanentDeleteTrash)
	authMux.HandleFunc("/api/files/trash/clear", h.clearTrash)

	// 快速访问管理
	authMux.HandleFunc("/api/files/quick-dirs/add", h.addQuickDir)
	authMux.HandleFunc("/api/files/quick-dirs/remove", h.removeQuickDir)
	authMux.HandleFunc("/api/files/quick-dirs/update", h.updateQuickDir)

	// 日志管理
	authMux.HandleFunc("/api/logs", h.handleLogsList)
	authMux.HandleFunc("/api/logs/read", h.handleLogsRead)
	authMux.HandleFunc("/api/logs/stats", h.handleLogsStats)
	authMux.HandleFunc("/api/logs/download", h.handleLogsDownload)
	authMux.HandleFunc("/api/logs/clear", h.handleLogsClear)
	authMux.HandleFunc("/api/logs/bulk-clear", h.handleLogsBulkClear)
	authMux.HandleFunc("/api/logs/bulk-export", h.handleLogsBulkExport)
	authMux.HandleFunc("/api/logs/config", h.handleLogsConfig)

	// FTP 服务控制
	authMux.HandleFunc("/api/service/ftp/toggle", h.toggleFTP)
	authMux.HandleFunc("/api/service/ftp/start", h.startFTP)
	authMux.HandleFunc("/api/service/ftp/stop", h.stopFTP)
	authMux.HandleFunc("/api/service/ftp/restart", h.restartFTP)
	authMux.HandleFunc("/api/service/ftp/reload", h.reloadFTP)

	// FTP 文件管理
	authMux.HandleFunc("/api/ftp/files", h.listFtpFiles)
	authMux.HandleFunc("/api/ftp/files/upload", h.uploadFtpFile)
	authMux.HandleFunc("/api/ftp/files/delete", h.deleteFtpFile)
	authMux.HandleFunc("/api/ftp/files/mkdir", h.mkdirFtp)
	authMux.HandleFunc("/api/ftp/files/download", h.downloadFtpFile)
	authMux.HandleFunc("/api/ftp/files/rename", h.renameFtpFile)
	authMux.HandleFunc("/api/ftp/files/copy", h.copyFtpFile)

	// FTP 用户管理
	authMux.HandleFunc("/api/ftp/users", h.listFtpUsers)
	authMux.HandleFunc("/api/ftp/users/add", h.addFtpUser)
	authMux.HandleFunc("/api/ftp/users/delete", h.deleteFtpUser)
	authMux.HandleFunc("/api/ftp/users/toggle", h.toggleFtpUserStatus)
	authMux.HandleFunc("/api/ftp/users/batch", h.batchFtpUsers)
	authMux.HandleFunc("/api/ftp/status", h.getFtpStatus)
	authMux.HandleFunc("/api/ftp/port", h.saveFtpPort)
	authMux.HandleFunc("/api/ftp/users/{id}", h.handleFtpUserDetail)

	// 站点管理
	authMux.HandleFunc("/api/sites", h.handleSitesList)
	authMux.HandleFunc("/api/sites/toggle", h.handleSiteToggle)
	authMux.HandleFunc("/api/sites/start", h.handleSiteStart)
	authMux.HandleFunc("/api/sites/stop", h.handleSiteStop)
	authMux.HandleFunc("/api/sites/restart", h.handleSiteRestart)
	authMux.HandleFunc("/api/sites/batch", h.handleSitesBatch)
	authMux.HandleFunc("/api/sites/status", h.getSitesStatus)
	authMux.HandleFunc("/api/sites/{id}", h.handleSitesDetail)

	// 站点服务控制
	authMux.HandleFunc("/api/service/sites/toggle", h.toggleSitesService)
	authMux.HandleFunc("/api/service/sites/start", h.startSitesService)
	authMux.HandleFunc("/api/service/sites/stop", h.stopSitesService)
	authMux.HandleFunc("/api/service/sites/restart", h.restartSitesService)
	authMux.HandleFunc("/api/service/sites/reload", h.reloadSitesConfig)

	// SSL 证书管理
	authMux.HandleFunc("/api/certs", h.handleCertsList)
	authMux.HandleFunc("/api/certs/request", h.handleCertRequest)
	authMux.HandleFunc("/api/certs/renew", h.handleCertRenew)
	authMux.HandleFunc("/api/certs/upload", h.handleCertUpload)
	authMux.HandleFunc("/api/certs/delete", h.handleCertDelete)
	authMux.HandleFunc("/api/certs/paste", h.handleCertPaste)
	authMux.HandleFunc("/api/certs/deploy", h.handleCertDeploy)
	authMux.HandleFunc("/api/certs/dns-prepare", h.handleCertDNSPrepare)
	authMux.HandleFunc("/api/certs/dns-complete", h.handleCertDNSComplete)
	authMux.HandleFunc("/api/certs/file-prepare", h.handleCertFilePrepare)
	authMux.HandleFunc("/api/certs/file-complete", h.handleCertFileComplete)
	authMux.HandleFunc("/api/certs/dns-providers", h.handleDNSProviders)
	authMux.HandleFunc("/api/certs/dns-providers/{id}", h.handleDNSProvidersRoute)
	authMux.HandleFunc("/api/certs/dns-providers/{id}/test", h.handleDNSProviderTest)
	authMux.HandleFunc("/api/certs/dns-providers/{id}/credentials", h.handleDNSProviderGetCreds)
	authMux.HandleFunc("/api/certs/dns-providers-test", h.handleDNSProviderTestCreds)
	authMux.HandleFunc("/api/certs/progress/{id}", h.handleCertProgress)

	// Web 终端
	authMux.HandleFunc("/api/terminal/ws", h.handleTerminalWS)

	// 开机自启
	authMux.HandleFunc("/api/service/autostart/status", h.getAutoStartStatus)
	authMux.HandleFunc("/api/service/autostart/enable", h.enableAutoStart)
	authMux.HandleFunc("/api/service/autostart/disable", h.disableAutoStart)

	authMux.HandleFunc("/", h.indexPage)
	// 将认证路由组挂载
	mux.Handle("/", h.RequireAuth(h.CSRPMiddleware(authMux)))

	return Chain(RecoveryMiddleware, LoggingMiddleware, SecurityHeadersMiddleware)(mux)
}

// routerCache 缓存的路由实例
var (
	routerCache http.Handler
	routerOnce  sync.Once
)

// getRouter 获取路由实例（懒初始化，并发安全）
func (h *Handler) getRouter() http.Handler {
	routerOnce.Do(func() {
		routerCache = h.setupRouter()
	})
	return routerCache
}
