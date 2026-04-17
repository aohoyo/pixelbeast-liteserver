package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pixelbeast/src/panel"
	"pixelbeast/src/config"
	"pixelbeast/src/site"
	embedfs "pixelbeast/src"
	"pixelbeast/src/ftp"
	"pixelbeast/src/logger"
	"pixelbeast/src/ssl"
	"pixelbeast/src/file"
)

var (
	version   = "0.1.0-dev"
	buildTime = "unknown"
)


func main() {
	configDir := flag.String("config", "config", "配置目录路径")
	showVersion := flag.Bool("version", false, "显示版本信息")
	flag.Parse()

	if *showVersion {
			logger.LogPanelSystem(logger.LogLevelInfo, "v%s (build: %s)", version, buildTime)
		return
	}

	// 加载配置
	cm, err := config.NewConfigManager(*configDir)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	serverName := cm.Server.Name
	if serverName == "" {
		serverName = "PixelBeast Server"
	}

	// 初始化日志
	logCfg := cm.Server.Log
	if err := logger.InitLoggerWithConfig("./log", &config.LogConfig{
		RetentionDays: logCfg.RetentionDays,
		MaxSizeMB:     logCfg.MaxSizeMB,
		CompressDays:  logCfg.CompressDays,
		CleanupHour:   logCfg.CleanupHour,
		Level:         logCfg.Level,
	}); err != nil {
		logger.LogPanelSystem(logger.LogLevelWarn, "警告: 初始化日志失败: %v", err)
	}

	logger.LogPanelSystem(logger.LogLevelInfo, "🦖 %s v%s 启动中...", serverName, version)

	// SSL 管理器（独立）
	sslMgr := ssl.NewSSLManager("./ssl")
	if err := sslMgr.Start(); err != nil {
		logger.LogPanelSystem(logger.LogLevelWarn, "SSL管理器启动失败: %v", err)
	}
	sslMgr.LoadSiteCertificates(cm.Sites.Sites)

	// 文件管理器（独立）
	fileMgr := file.NewFileManager()
	fileMgr.UpdateBookmarksFromConfig(cm.Sites.Sites, cm.GetSitesDir())

	// 站点管理器（独立）
	siteMgr := site.NewSiteManager(cm, sslMgr, fileMgr)

	// FTP 服务器（独立）
	var ftpSrv *ftp.FTPServer
	ftpCfg := cm.FTP
	if ftpCfg.Port > 0 {
		ftpSrv, err = ftp.NewFTPServerWithValidator(ftpCfg, cm, cm.GetFTPRoot())
		if err != nil {
			logger.LogPanelSystem(logger.LogLevelWarn, "创建FTP服务器失败: %v", err)
		}
	}

	// 设置静态文件系统
	panel.SetStaticFS(embedfs.GetStaticFS())

	// 创建管理面板处理器
	adminHandler := panel.New(cm, *configDir)
	adminHandler.Version = version
	adminHandler.SetSiteManager(siteMgr)
	adminHandler.FTPServer = ftpSrv

	// DNS 自动申请证书成功后，自动更新站点 SSL 配置
	sslMgr.SetOnCertObtained(func(domain, provider, challengeMethod, email string) {
		adminHandler.UpdateSiteSSLConfig(domain, &config.SSLConfig{
			Enabled:         true,
			AutoHTTPS:       true,
			Provider:        provider,
			ChallengeMethod: challengeMethod,
			Email:           email,
		})
	})

	// 启动管理面板服务器（独立）
	adminPort := cm.Server.Admin.Port
	if adminPort <= 0 || adminPort > 65535 {
		adminPort = 9527
	}
	adminServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", adminPort),
		Handler:      adminHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	go func() {
		if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.LogPanelSystem(logger.LogLevelError, "[Admin] 服务错误: %v", err)
		}
	}()

	// 启动网站服务器
	if err := siteMgr.StartSitesServer(); err != nil {
		logger.LogPanelSystem(logger.LogLevelWarn, "启动网站服务器失败: %v", err)
	}

	// 启动 FTP 服务
	if ftpCfg.Enabled && ftpSrv != nil {
		if err := ftpSrv.Start(); err != nil {
			logger.LogPanelSystem(logger.LogLevelError, "FTP服务器启动失败: %v", err)
		} else {
			adminHandler.FTPServer = ftpSrv // 已在上方设置，确保引用一致
			adminHandler.SetFTPRunning(true)
		}
	}

	logger.LogPanelSystem(logger.LogLevelInfo, "%s 启动完成", serverName)
	fmt.Printf("\n  \033[32;1m  🦖 管理面板: http://localhost:%d/admin\033[0m\n\n", adminPort)

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	logger.LogPanelSystem(logger.LogLevelInfo, "收到信号 %v，正在关闭...", sig)

	if ftpSrv != nil {
		ftpSrv.Stop()
	}
	if siteMgr.IsSitesRunning() {
		siteMgr.StopSitesServer()
	}
	sslMgr.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	adminServer.Shutdown(ctx)

	logger.LogPanelSystem(logger.LogLevelInfo, "%s 已关闭", serverName)
	logger.Close()
}
