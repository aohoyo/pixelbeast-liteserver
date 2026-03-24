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
	configPath := flag.String("config", "pixelbeast.json", "配置文件路径")
	showVersion := flag.Bool("version", false, "显示版本信息")
	flag.Parse()

	if *showVersion {
		fmt.Printf("像素兽 (PixelBeast) v%s\n", version)
		fmt.Printf("构建时间: %s\n", buildTime)
		return
	}

	printBanner()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建默认目录
	if err := cfg.CreateDefaultDirectories(); err != nil {
		log.Fatalf("创建目录失败: %v", err)
	}

	// 初始化日志
	if err := handlers.InitLogger("./logs"); err != nil {
		log.Printf("警告: 初始化日志失败: %v", err)
	}

	log.Printf("配置文件: %s", *configPath)

	// 创建服务管理器
	serverManager = handlers.NewServerManager(cfg, *configPath)

	// 创建 FTP 服务器
	if cfg.FTP.Port > 0 {
		ftpServer, err := handlers.NewFTPServer(&cfg.FTP)
		if err != nil {
			log.Printf("警告: 创建FTP服务器失败: %v", err)
		} else {
			serverManager.SetFTPServer(ftpServer, &cfg.FTP)
		}
	}

	// 设置静态文件系统
	admin.SetStaticFS(getStaticFS())

	// 创建管理面板处理器
	adminHandler := admin.New(cfg, *configPath)
	adminHandler.SetServerManager(serverManager)
	serverManager.SetAdminHandler(adminHandler)

	// 启动管理面板服务器
	if err := serverManager.StartAdminPanel(); err != nil {
		log.Fatalf("启动管理面板失败: %v", err)
	}

	// 启动网站服务器
	if err := serverManager.StartSitesServer(); err != nil {
		log.Printf("警告: 启动网站服务器失败: %v", err)
	}

	// 启动 FTP 服务（根据配置）
	if cfg.FTP.Enabled {
		if err := serverManager.StartFTP(); err != nil {
			log.Printf("FTP服务器启动失败: %v", err)
		} else {
			log.Printf("FTP服务器启动在端口 %d", cfg.FTP.Port)
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
