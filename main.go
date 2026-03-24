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
	version   = "3.0.0"
	buildTime = "unknown"
)

var serverManager *handlers.ServerManager

func main() {
	configDir := flag.String("config", "config", "配置目录路径")
	showVersion := flag.Bool("version", false, "显示版本信息")
	flag.Parse()

	if *showVersion {
		fmt.Printf("像素兽 (PixelBeast) v%s\n", version)
		fmt.Printf("构建时间: %s\n", buildTime)
		return
	}

	printBanner()

	// 加载配置
	cm, err := config.NewConfigManager(*configDir)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	adapter := config.NewAdapter(cm)

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

	log.Printf("配置目录: %s", *configDir)

	// 创建服务管理器（临时用旧配置结构，后续重构）
	tmpCfg := &config.Config{
		Global: config.GlobalConfig{
			AdminPort: adapter.GetAdminPort(),
		},
	}
	serverManager = handlers.NewServerManager(tmpCfg, *configDir)

	// 创建 FTP 服务器
	ftpCfg := adapter.GetFTP()
	if ftpCfg.Port > 0 {
		ftpServer, err := handlers.NewFTPServerWithValidator(ftpCfg, cm)
		if err != nil {
			log.Printf("警告: 创建FTP服务器失败: %v", err)
		} else {
			serverManager.SetFTPServer(ftpServer, ftpCfg)
		}
	}

	// 设置静态文件系统
	admin.SetStaticFS(getStaticFS())

	// 创建管理面板处理器
	tmpAdminCfg := &config.Config{
		Admin: config.AdminConfig{
			Username: cm.Server.AdminUsername,
			Path:     cm.Server.AdminPath,
		},
	}
	adminHandler := admin.New(tmpAdminCfg, *configDir)
	adminHandler.SetServerManager(serverManager)
	adminHandler.SetPasswordValidator(cm)
	serverManager.SetAdminHandler(adminHandler)

	// 启动管理面板服务器
	if err := serverManager.StartAdminPanel(); err != nil {
		log.Fatalf("启动管理面板失败: %v", err)
	}

	// 启动网站服务器
	if err := serverManager.StartSitesServer(); err != nil {
		log.Printf("警告: 启动网站服务器失败: %v", err)
	}

	// 启动 FTP 服务
	if ftpCfg.Enabled {
		if err := serverManager.StartFTP(); err != nil {
			log.Printf("FTP服务器启动失败: %v", err)
		} else {
			log.Printf("FTP服务器启动在端口 %d", ftpCfg.Port)
		}
	}

	// 优雅关闭
	setupGracefulShutdown()

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}

func printBanner() {
	fmt.Println()
	fmt.Println("  🪶 像素兽 PixelBeast v" + version)
	fmt.Println("  小而强悍，无所不能 - 多站点版")
	fmt.Println()
}

func setupGracefulShutdown() {
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("正在关闭服务器...")

		// 停止 FTP
		if serverManager.IsFTPRunning() {
			serverManager.StopFTP()
		}

		// 停止网站服务器
		if serverManager.IsSitesRunning() {
			serverManager.StopSitesServer()
		}

		// 停止管理面板
		if serverManager.IsAdminRunning() {
			serverManager.StopAdminPanel()
		}

		log.Println("服务器已关闭")
		os.Exit(0)
	}()
}