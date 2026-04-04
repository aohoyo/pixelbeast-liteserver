package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"pixelbeast/src/admin"
	"pixelbeast/src/config"
	"pixelbeast/src/handlers"
)

var (
	version   = "3.1.4"
	buildTime = "unknown"
)

var serverManager *handlers.ServerManager

func main() {
	configDir := flag.String("config", "config", "配置目录路径")
	showVersion := flag.Bool("version", false, "显示版本信息")
	flag.Parse()

	if *showVersion {
		fmt.Printf("v%s (build: %s)\n", version, buildTime)
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
	if err := handlers.InitLoggerWithConfig("./log", &config.LogConfig{
		RetentionDays: logCfg.RetentionDays,
		MaxSizeMB:     logCfg.MaxSizeMB,
		CompressDays:  logCfg.CompressDays,
		CleanupHour:   logCfg.CleanupHour,
		Level:         logCfg.Level,
	}); err != nil {
		log.Printf("警告: 初始化日志失败: %v", err)
	}

	handlers.LogSystemInfo("🪶 %s v%s 启动中...", serverName, version)

	// 创建服务管理器
	serverManager = handlers.NewServerManager(cm, *configDir)

	// 创建 FTP 服务器
	ftpCfg := cm.FTP
	if ftpCfg.Port > 0 {
		ftpServer, err := handlers.NewFTPServerWithValidator(ftpCfg, cm, cm.GetFTPRoot())
		if err != nil {
			handlers.LogSystemWarn("创建FTP服务器失败: %v", err)
		} else {
			serverManager.SetFTPServer(ftpServer, ftpCfg)
		}
	}

	// 设置静态文件系统
	admin.SetStaticFS(getStaticFS())

	// 创建管理面板处理器
	adminHandler := admin.New(cm, *configDir)
	adminHandler.Version = version
	adminHandler.SetServerManager(serverManager)
	serverManager.SetAdminHandler(adminHandler)

	// 启动管理面板服务器
	if err := serverManager.StartAdminPanel(); err != nil {
		handlers.LogSystemError("启动管理面板失败: %v", err)
		log.Fatalf("启动管理面板失败: %v", err)
	}

	// 启动网站服务器
	if err := serverManager.StartSitesServer(); err != nil {
		handlers.LogSystemWarn("启动网站服务器失败: %v", err)
	}

	// 启动 FTP 服务
	if ftpCfg.Enabled {
		if err := serverManager.StartFTP(); err != nil {
			handlers.LogSystemError("FTP服务器启动失败: %v", err)
		}
	}

	handlers.LogSystemInfo("%s 启动完成", serverName)

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	handlers.LogSystemInfo("收到信号 %v，正在关闭...", sig)

	if serverManager.IsFTPRunning() {
		serverManager.StopFTP()
	}
	if serverManager.IsSitesRunning() {
		serverManager.StopSitesServer()
	}
	if serverManager.IsAdminRunning() {
		serverManager.StopAdminPanel()
	}

	handlers.LogSystemInfo("%s 已关闭", serverName)
	handlers.Close()
}
