package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"pixelbeast/src/config"
	"pixelbeast/src/handlers"
)

var startTime = time.Now()

// cpuHistory 存储 CPU 历史数据用于趋势图
var cpuHistory []float64

// ==================== 状态 ====================

func (h *Handler) getStatus(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	username := h.getSessionUsername(r)
	clientIP := getClientIP(r)
	defer func() {
		handlers.LogPanelAPI(username, r.Method, r.URL.Path, clientIP, 200, time.Since(start))
	}()

	processMem := handlers.GetProcessMemory()

	// 获取服务器状态
	adminRunning := false
	sitesRunning := false
	ftpRunning := h.Config.FTP.Enabled
	adminPort := h.Config.Global.AdminPort

	if h.ServerManager != nil {
		adminRunning = h.ServerManager.IsAdminRunning()
		sitesRunning = h.ServerManager.IsSitesRunning()
		ftpRunning = h.ServerManager.IsFTPRunning()
	}

	// 构建站点列表
	sites := make([]map[string]interface{}, 0)
	for _, site := range h.Config.Sites {
		sites = append(sites, map[string]interface{}{
			"id":      site.ID,
			"name":    site.Name,
			"enabled": site.Enabled,
			"type":    site.Type,
			"port":    site.Port,
			"domains": site.Domain,
		})
	}

	session := h.getSession(r)
	var csrfToken string
	if session != nil {
		if cookie, _ := r.Cookie("session_id"); cookie != nil {
			csrfToken = h.generateCSRFToken(cookie.Value)
		}
	}

	// 构建状态数据
	statusData := map[string]interface{}{
		"memory_mb":         processMem / 1024 / 1024,
		"goroutines":        runtime.NumGoroutine(),
		"go_version":        runtime.Version(),
		"os":                runtime.GOOS,
		"arch":              runtime.GOARCH,
		"server_start_time": startTime.UnixMilli(), // 返回启动时间戳（新方案）
		"uptime":            time.Since(startTime).Milliseconds(), // 保留兼容旧代码
		"admin_running":     adminRunning,
		"admin_port":        adminPort,
		"sites_running":     sitesRunning,
		"sites_count":       len(h.Config.Sites),
		"sites":             sites,
		"ftp_running":       ftpRunning,
		"ftp_port":          h.Config.FTP.Port,
		"ftp_root":          h.Config.FTP.Root,
		"ftp_dir":           h.Config.Global.FTPDir,
		"backup_dir":        h.Config.Global.BackupDir,
	}

	// 如果有 CSRF token，添加到响应中
	if csrfToken != "" {
		statusData["csrf_token"] = csrfToken
	}

	Success(w, statusData)
}

// ==================== 系统监控 API ====================

func (h *Handler) getSystemStatus(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	username := h.getSessionUsername(r)
	clientIP := getClientIP(r)
	defer func() {
		handlers.LogPanelAPI(username, r.Method, r.URL.Path, clientIP, 200, time.Since(start))
	}()

	// 获取进程内存
	processMem := handlers.GetProcessMemory()
	memoryMB := float64(processMem) / 1024 / 1024

	// 模拟系统数据（后期实现真实数据获取）
	// TODO: 使用 gopsutil 等库获取真实的 CPU、内存、硬盘、负载数据
	cpuPercent := getSimulatedCPUPercent()
	cpuCores := runtime.NumCPU()

	// 更新 CPU 历史记录
	cpuHistory = append(cpuHistory, cpuPercent)
	if len(cpuHistory) > 5 {
		cpuHistory = cpuHistory[len(cpuHistory)-5:]
	}

	// 模拟数据（后续替换为真实系统数据）
	statusData := map[string]interface{}{
		// CPU
		"cpu_percent": cpuPercent,
		"cpu_cores":   cpuCores,
		"cpu_threads": cpuCores * 2,
		"cpu_history": cpuHistory,

		// 内存（模拟数据）
		"memory_percent": 40.0,
		"memory_used_gb": 3.2,
		"memory_total_gb": 8.0,

		// 硬盘（模拟数据）
		"disk_percent": 45.0,
		"disk_used_gb": 45.0,
		"disk_total_gb": 100.0,

		// 负载（模拟数据）
		"load_avg":      []float64{0.52, 0.48, 0.45},
		"process_count": 128,
		"thread_count":  256,

		// 运行时间（返回启动时间戳，前端自行计算）
		"server_start_time": startTime.UnixMilli(),

		// 保留原有字段
		"memory_mb":  memoryMB,
		"goroutines": runtime.NumGoroutine(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
	}

	Success(w, statusData)
}

// getSimulatedCPUPercent 返回模拟的 CPU 使用率（后期替换为真实数据）
func getSimulatedCPUPercent() float64 {
	// 基于时间产生波动的模拟数据
	now := time.Now().Unix()
	base := 25.0
	fluctuation := float64(now%30) * 1.5
	result := base + fluctuation
	if result > 80 {
		result = 80
	}
	return result
}

func (h *Handler) freeMemory(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	username := h.getSessionUsername(r)
	clientIP := getClientIP(r)
	defer func() {
		handlers.LogPanelAPI(username, r.Method, r.URL.Path, clientIP, 200, time.Since(start))
	}()

	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	// TODO: 实现真实的内存释放逻辑
	// 目前返回模拟数据
	runtime.GC() // 触发 Go 的垃圾回收

	// 计算释放的内存（模拟）
	beforeMem := handlers.GetProcessMemory()
	runtime.GC()
	time.Sleep(100 * time.Millisecond)
	afterMem := handlers.GetProcessMemory()
	freedMB := float64(beforeMem-afterMem) / 1024 / 1024
	if freedMB < 0 {
		freedMB = 0
	}

	Success(w, map[string]interface{}{
		"freed_mb": freedMB,
		"message":  fmt.Sprintf("已释放 %.1f MB 内存", freedMB),
	})
}

func (h *Handler) scanCleanup(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	username := h.getSessionUsername(r)
	clientIP := getClientIP(r)
	defer func() {
		handlers.LogPanelAPI(username, r.Method, r.URL.Path, clientIP, 200, time.Since(start))
	}()

	if r.Method != http.MethodGet {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	// TODO: 实现真实的垃圾扫描逻辑
	// 扫描日志文件、临时文件等
	logsMB := 0.0
	tempMB := 0.0

	// 扫描日志目录
	logDir := handlers.GetLogBaseDir()
	if logDir != "" {
		entries, err := os.ReadDir(logDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					info, err := entry.Info()
					if err == nil {
						logsMB += float64(info.Size()) / 1024 / 1024
					}
				}
			}
		}
	}

	totalMB := logsMB + tempMB

	Success(w, map[string]interface{}{
		"logs_mb":  logsMB,
		"temp_mb":  tempMB,
		"total_mb": totalMB,
	})
}

func (h *Handler) executeCleanup(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	username := h.getSessionUsername(r)
	clientIP := getClientIP(r)
	defer func() {
		handlers.LogPanelAPI(username, r.Method, r.URL.Path, clientIP, 200, time.Since(start))
	}()

	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	// TODO: 实现真实的清理逻辑
	// 清理旧的日志文件、临时文件等

	// 返回模拟的清理结果
	Success(w, map[string]interface{}{
		"cleaned_mb": 128.5,
		"message":    "已清理 128.5 MB",
	})
}

// ==================== 配置 ====================

func (h *Handler) getConfig(w http.ResponseWriter, r *http.Request) {
	Success(w, h.Config)
}

func (h *Handler) saveConfig(w http.ResponseWriter, r *http.Request) {
	username := h.getSessionUsername(r)
	if r.Method != http.MethodPost {
		BadRequest(w, "Method not allowed")
		handlers.LogPanelConfigChange(username, "保存配置", false)
		return
	}
	var newConfig config.Config
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		BadRequest(w, "Invalid JSON")
		handlers.LogPanelConfigChange(username, "保存配置", false)
		return
	}
	if err := newConfig.Save(h.ConfigPath); err != nil {
		InternalServerError(w, err.Error())
		handlers.LogPanelConfigChange(username, "保存配置", false)
		return
	}
	*h.Config = newConfig
	handlers.LogPanelConfigChange(username, "保存配置", true)
	SuccessMessage(w, "配置已保存")
}

// ==================== 日志 ====================

func (h *Handler) getLogs(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	logType := r.URL.Query().Get("type")
	lines, _ := strconv.Atoi(r.URL.Query().Get("lines"))
	if lines == 0 {
		lines = 100
	}

	// 兼容旧的单层参数
	if category == "" {
		if logType == "access" {
			category = "http"
		} else if logType == "error" {
			category = "http"
		} else if logType == "server" {
			category = "http"
		} else {
			category = "http"
			logType = "access"
		}
	}
	if logType == "" {
		logType = "access"
	}

	// 安全检查：防止路径遍历
	if strings.Contains(category, "..") || strings.Contains(logType, "..") {
		BadRequest(w, "Invalid log type")
		return
	}

	// 构建日志文件路径
	logPath := handlers.GetLogFilePath(category, logType)
	if logPath == "" {
		BadRequest(w, "Invalid log type")
		return
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		Success(w, map[string]interface{}{"logs": []string{}})
		return
	}
	allLines := strings.Split(string(data), "\n")
	start := 0
	if len(allLines) > lines {
		start = len(allLines) - lines
	}
	Success(w, map[string]interface{}{"logs": allLines[start:]})
}

func (h *Handler) clearLogs(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	logType := r.URL.Query().Get("type")

	// 兼容旧的单层参数
	if category == "" {
		if logType == "access" {
			category = "http"
		} else if logType == "error" {
			category = "http"
		} else if logType == "server" {
			category = "http"
		} else {
			category = "http"
			logType = "access"
		}
	}
	if logType == "" {
		logType = "access"
	}

	// 安全检查：防止路径遍历
	if strings.Contains(category, "..") || strings.Contains(logType, "..") {
		BadRequest(w, "Invalid log type")
		return
	}

	// 构建日志文件路径
	logPath := handlers.GetLogFilePath(category, logType)
	if logPath == "" {
		BadRequest(w, "Invalid log type")
		return
	}

	os.WriteFile(logPath, []byte{}, 0644)
	SuccessMessage(w, "日志已清空")
}
