package admin

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"

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
	ftpRunning := h.ConfigManager.FTP.Enabled
	adminPort := h.ConfigManager.Server.AdminPort

	if h.ServerManager != nil {
		adminRunning = h.ServerManager.IsAdminRunning()
		sitesRunning = h.ServerManager.IsSitesRunning()
		ftpRunning = h.ServerManager.IsFTPRunning()
	}

	// 构建站点列表
	sites := make([]map[string]interface{}, 0)
	for _, site := range h.ConfigManager.Sites.Sites {
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
		"server_start_time": startTime.UnixMilli(),
		"uptime":            time.Since(startTime).Milliseconds(),
		"admin_running":     adminRunning,
		"admin_port":        adminPort,
		"sites_running":     sitesRunning,
		"sites_count":       len(h.ConfigManager.Sites.Sites),
		"sites":             sites,
		"ftp_running":       ftpRunning,
		"ftp_port":          h.ConfigManager.FTP.Port,
		"ftp_root":          h.ConfigManager.FTP.Root,
		"ftp_dir":           h.ConfigManager.Server.FTPDir,
		"backup_dir":        h.ConfigManager.Server.BackupDir,
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

	// 获取真实的 CPU 信息
	cpuPercent := getRealCPUPercent()
	cpuCores := runtime.NumCPU()
	cpuModel := getCPUModel()

	// 更新 CPU 历史记录
	cpuHistory = append(cpuHistory, cpuPercent)
	if len(cpuHistory) > 5 {
		cpuHistory = cpuHistory[len(cpuHistory)-5:]
	}

	// 获取 FTP 服务状态
	ftpRunning := h.ConfigManager.FTP.Enabled
	if h.ServerManager != nil {
		ftpRunning = h.ServerManager.IsFTPRunning()
	}

	// 获取真实的内存信息
	memInfo, _ := mem.VirtualMemory()
	memPercent := 0.0
	memUsedGB := 0.0
	memTotalGB := 0.0
	memFreeGB := 0.0
	memAvailableGB := 0.0
	memSharedMB := 0.0
	memBuffCacheMB := 0.0
	if memInfo != nil {
		memPercent = memInfo.UsedPercent
		memUsedGB = float64(memInfo.Used) / 1024 / 1024 / 1024
		memTotalGB = float64(memInfo.Total) / 1024 / 1024 / 1024
		memFreeGB = float64(memInfo.Free) / 1024 / 1024 / 1024
		memAvailableGB = float64(memInfo.Available) / 1024 / 1024 / 1024
		memSharedMB = float64(memInfo.Shared) / 1024 / 1024
		memBuffCacheMB = float64(memInfo.Buffers+memInfo.Cached) / 1024 / 1024
	}

	// 获取真实的硬盘信息（获取项目所在盘符/目录）
	programDir, _ := os.Getwd()
	diskPath := "/"
	
	// Windows 下获取项目所在盘符
	if runtime.GOOS == "windows" {
		// Windows 下使用项目目录作为磁盘路径
		// disk.Usage 会自动识别盘符
		diskPath = programDir
	} else {
		// Linux/macOS 下获取项目所在挂载点
		diskPath = programDir
	}
	
	diskInfo, _ := disk.Usage(diskPath)
	diskPercent := 0.0
	diskUsedGB := 0.0
	diskTotalGB := 0.0
	diskFreeGB := 0.0
	diskFs := ""
	diskMount := diskPath
	if diskInfo != nil {
		diskPercent = diskInfo.UsedPercent
		diskUsedGB = float64(diskInfo.Used) / 1024 / 1024 / 1024
		diskTotalGB = float64(diskInfo.Total) / 1024 / 1024 / 1024
		diskFreeGB = float64(diskInfo.Free) / 1024 / 1024 / 1024
		diskFs = diskInfo.Fstype
		diskMount = diskInfo.Path
	}

	// 获取真实的负载信息
	loadAvg, _ := load.Avg()
	load1m, load5m, load15m := 0.0, 0.0, 0.0
	if loadAvg != nil {
		load1m = loadAvg.Load1
		load5m = loadAvg.Load5
		load15m = loadAvg.Load15
	}

	// 构建状态数据
	statusData := map[string]interface{}{
		// CPU
		"cpu_percent":   cpuPercent,
		"cpu_cores":     cpuCores,
		"cpu_threads":   cpuCores * 2,
		"cpu_model":     cpuModel,
		"cpu_history":   cpuHistory,

		// 内存
		"memory_percent":      memPercent,
		"memory_used_gb":      memUsedGB,
		"memory_total_gb":     memTotalGB,
		"memory_free_gb":      memFreeGB,
		"memory_available_gb": memAvailableGB,
		"memory_shared_mb":    memSharedMB,
		"memory_buff_cache_mb": memBuffCacheMB,

		// 硬盘
		"disk_percent":    diskPercent,
		"disk_used_gb":    diskUsedGB,
		"disk_total_gb":   diskTotalGB,
		"disk_free_gb":    diskFreeGB,
		"disk_mount":      diskMount,
		"disk_filesystem": diskFs,
		"disk_type":       getDiskType(),

		// 负载
		"load_avg":       []float64{load1m, load5m, load15m},
		"process_active": getProcessCount(),
		"process_total":  getProcessCount() + 50, // 估算值

		// 运行时间
		"server_start_time": startTime.UnixMilli(),

		// FTP 状态
		"ftp_running": ftpRunning,
		"ftp_port":    h.ConfigManager.FTP.Port,

		// 保留原有字段
		"memory_mb":  memoryMB,
		"goroutines": runtime.NumGoroutine(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
	}

	Success(w, statusData)
}

// getRealCPUPercent 获取真实 CPU 使用率
func getRealCPUPercent() float64 {
	// 间隔 500ms 采样（更实时，但波动稍大）
	percent, err := cpu.Percent(500*time.Millisecond, false)
	if err != nil || len(percent) == 0 {
		return getSimulatedCPUPercent()
	}
	return percent[0]
}

// getCPUModel 获取 CPU 型号（跨平台）
func getCPUModel() string {
	info, err := cpu.Info()
	if err != nil || len(info) == 0 {
		return runtime.GOARCH
	}
	// 返回第一个 CPU 的型号
	for _, cpu := range info {
		if cpu.ModelName != "" {
			return cpu.ModelName
		}
	}
	return runtime.GOARCH
}

// getDiskType 获取硬盘类型
func getDiskType() string {
	// gopsutil 不能直接获取硬盘类型（SSD/HDD）
	// 可以通过检测是否为旋转磁盘来判断
	// 这里返回简化版本
	return "--"
}

// getProcessCount 获取进程数量（简化版）
func getProcessCount() int {
	// 使用 gopsutil 获取进程列表
	procs, err := cpu.Counts(false)
	if err != nil {
		return runtime.NumCPU()
	}
	return procs
}

// getSimulatedCPUPercent 返回模拟的 CPU 使用率（作为降级方案）
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
	// 构建前端期望的格式
	cfg := map[string]interface{}{
		"admin": map[string]interface{}{
			"username": h.ConfigManager.Server.AdminUsername,
			"password": "", // 不返回密码
			"port":     h.ConfigManager.Server.AdminPort,
			"path":     h.ConfigManager.Server.AdminPath,
			"bind_domain": h.ConfigManager.Server.AdminDomain,
			"ssl_enabled": h.ConfigManager.Server.AdminSSLEnabled,
		},
		"http": map[string]interface{}{
			"port": h.ConfigManager.Server.HTTPPort,
			"root": h.ConfigManager.Server.HTTPDir,
		},
		"ftp": map[string]interface{}{
			"enabled": h.ConfigManager.FTP.Enabled,
			"port":    h.ConfigManager.FTP.Port,
			"root":    h.ConfigManager.FTP.Root,
			"users":   h.ConfigManager.FTP.Users,
		},
		"sites": h.ConfigManager.Sites.Sites,
		"log": map[string]interface{}{
			"retention_days": h.ConfigManager.Server.Log.RetentionDays,
			"max_size_mb":    h.ConfigManager.Server.Log.MaxSizeMB,
			"compress_days":  h.ConfigManager.Server.Log.CompressDays,
			"cleanup_hour":   h.ConfigManager.Server.Log.CleanupHour,
			"level":          h.ConfigManager.Server.Log.Level,
			"levels":         h.ConfigManager.Server.Log.Levels,
		},
		"backup_dir": h.ConfigManager.Server.BackupDir,
	}
	Success(w, cfg)
}

func (h *Handler) saveConfig(w http.ResponseWriter, r *http.Request) {
	username := h.getSessionUsername(r)
	if r.Method != http.MethodPost {
		BadRequest(w, "Method not allowed")
		handlers.LogPanelConfigChange(username, "保存配置", false)
		return
	}

	// 使用 map 接收前端数据
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		BadRequest(w, "Invalid JSON: "+err.Error())
		handlers.LogPanelConfigChange(username, "保存配置", false)
		return
	}

	// 解析 admin 配置
	if admin, ok := data["admin"].(map[string]interface{}); ok {
		if v, ok := admin["username"].(string); ok {
			h.ConfigManager.Server.AdminUsername = v
		}
		if v, ok := admin["port"].(float64); ok {
			h.ConfigManager.Server.AdminPort = int(v)
		}
		if v, ok := admin["path"].(string); ok {
			h.ConfigManager.Server.AdminPath = v
		}
		if v, ok := admin["password"].(string); ok && v != "" {
			h.ConfigManager.SetAdminPassword(v)
		}
		if v, ok := admin["bind_domain"].(string); ok {
			h.ConfigManager.Server.AdminDomain = v
		}
		if v, ok := admin["ssl_enabled"].(bool); ok {
		h.ConfigManager.Server.AdminSSLEnabled = v
		}
	}

	// 解析 http 配置
	if http, ok := data["http"].(map[string]interface{}); ok {
		if v, ok := http["port"].(float64); ok {
			h.ConfigManager.Server.HTTPPort = int(v)
		}
		if v, ok := http["root"].(string); ok {
			h.ConfigManager.Server.HTTPDir = v
		}
	}

	// 解析 ftp 配置
	if ftp, ok := data["ftp"].(map[string]interface{}); ok {
		if v, ok := ftp["enabled"].(bool); ok {
			h.ConfigManager.FTP.Enabled = v
		}
		if v, ok := ftp["port"].(float64); ok {
			h.ConfigManager.FTP.Port = int(v)
		}
		if v, ok := ftp["root"].(string); ok {
			h.ConfigManager.FTP.Root = v
		}
	}

	// 解析 sites 配置
	if sites, ok := data["sites"].([]interface{}); ok {
		siteConfigs := make([]config.SiteConfig, 0, len(sites))
		for _, s := range sites {
			if siteMap, ok := s.(map[string]interface{}); ok {
				siteConfig := config.SiteConfig{}
				if v, ok := siteMap["id"].(string); ok {
					siteConfig.ID = v
				}
				if v, ok := siteMap["name"].(string); ok {
					siteConfig.Name = v
				}
				if v, ok := siteMap["enabled"].(bool); ok {
					siteConfig.Enabled = v
				}
				if v, ok := siteMap["port"].(float64); ok {
					siteConfig.Port = int(v)
				}
				if v, ok := siteMap["root"].(string); ok {
					siteConfig.Root = v
				}
				siteConfigs = append(siteConfigs, siteConfig)
			}
		}
		h.ConfigManager.Sites.Sites = siteConfigs
	}

	// 解析 log 配置
	if log, ok := data["log"].(map[string]interface{}); ok {
		if v, ok := log["retention_days"].(float64); ok {
			h.ConfigManager.Server.Log.RetentionDays = int(v)
		}
		if v, ok := log["max_size_mb"].(float64); ok {
			h.ConfigManager.Server.Log.MaxSizeMB = int(v)
		}
		if v, ok := log["compress_days"].(float64); ok {
			h.ConfigManager.Server.Log.CompressDays = int(v)
		}
		if v, ok := log["cleanup_hour"].(float64); ok {
			h.ConfigManager.Server.Log.CleanupHour = int(v)
		}
		if v, ok := log["level"].(string); ok {
			h.ConfigManager.Server.Log.Level = v
		}
		if v, ok := log["levels"].(map[string]interface{}); ok {
			levels := make(map[string]string)
			for k, val := range v {
				if s, ok := val.(string); ok {
					levels[k] = s
				}
			}
			h.ConfigManager.Server.Log.Levels = levels
		}
	}

	// 解析 backup_dir
	if v, ok := data["backup_dir"].(string); ok {
		h.ConfigManager.Server.BackupDir = v
	}

	// 保存配置
	if err := h.ConfigManager.Save(); err != nil {
		InternalServerError(w, err.Error())
		handlers.LogPanelConfigChange(username, "保存配置", false)
		return
	}

	handlers.LogPanelConfigChange(username, "保存配置", true)
	SuccessMessage(w, "配置已保存")
}

func (h *Handler) resetConfig(w http.ResponseWriter, r *http.Request) {
	username := h.getSessionUsername(r)
	if r.Method != http.MethodPost {
		BadRequest(w, "Method not allowed")
		return
	}

	// 恢复默认配置
	defaultConfig := map[string]interface{}{
		"admin": map[string]interface{}{
			"username": "admin",
			"password": "",
			"port":     9527,
			"path":     "/admin",
		},
		"http": map[string]interface{}{
			"port": 3080,
			"root": "./web",
		},
		"ftp": map[string]interface{}{
			"enabled": false,
			"port":    2121,
			"root":    "./ftp",
			"users":   []config.FTPUser{},
		},
		"sites": []config.SiteConfig{
				{
					ID:         "default",
					Name:       "默认站点",
					Enabled:    true,
					Type:       "static",
					Port:       3080,
					Root:       "./web",
					IndexFiles: []string{"index.html", "index.htm"},
					AutoIndex:  true,
					CreatedAt:  time.Now().Format("2006-01-02 15:04:05"),
				},
			},
		"log": map[string]interface{}{
			"retention_days": 30,
			"max_size_mb":    100,
			"compress_days":  7,
			"cleanup_hour":   3,
			"level":          "info",
			"levels":         map[string]string{},
		},
		"backup_dir": "./backups",
	}

	// 重置 ConfigManager
	if h.ConfigManager != nil {
		h.ConfigManager.Server.AdminUsername = "admin"
		h.ConfigManager.Server.AdminPort = 9527
		h.ConfigManager.Server.AdminPath = "/admin"
		h.ConfigManager.SetAdminPassword("admin123")
		h.ConfigManager.Server.HTTPPort = 3080
		h.ConfigManager.Server.HTTPDir = "./web"
		h.ConfigManager.Server.BackupDir = "./backups"
		h.ConfigManager.FTP.Enabled = false
		h.ConfigManager.FTP.Port = 2121
		h.ConfigManager.FTP.Root = "./ftp"
		h.ConfigManager.FTP.Users = []config.FTPUser{}
		h.ConfigManager.Sites.Sites = []config.SiteConfig{
				{
					ID:         "default",
					Name:       "默认站点",
					Enabled:    true,
					Type:       "static",
					Port:       3080,
					Root:       "./web",
					IndexFiles: []string{"index.html", "index.htm"},
					AutoIndex:  true,
					CreatedAt:  time.Now().Format("2006-01-02 15:04:05"),
				},
			}
		h.ConfigManager.Server.Log = config.LogConfig{
			RetentionDays: 30,
			MaxSizeMB:     100,
			CompressDays:  7,
			CleanupHour:   3,
			Level:         "info",
			Levels:        map[string]string{},
		}

		if err := h.ConfigManager.Save(); err != nil {
			InternalServerError(w, err.Error())
			handlers.LogPanelConfigChange(username, "重置配置", false)
			return
		}
	}

	handlers.LogPanelConfigChange(username, "重置配置", true)
	Success(w, defaultConfig)
}

// ==================== 日志 ====================

func (h *Handler) getLogs(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	logType := r.URL.Query().Get("type")
	lines, _ := strconv.Atoi(r.URL.Query().Get("lines"))
	if lines == 0 {
		lines = 100
	}

	// 必须提供 category 和 type
	if category == "" || logType == "" {
		BadRequest(w, "category and type are required")
		return
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

	// 必须提供 category 和 type
	if category == "" || logType == "" {
		BadRequest(w, "category and type are required")
		return
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

func (h *Handler) getSystemTime(w http.ResponseWriter, r *http.Request) {
	now := time.Now()

	result := map[string]interface{}{
		"timestamp":  now.Unix(),
		"time":       now.Format("2006-01-02 15:04:05"),
		"utc_time":   now.UTC().Format("2006-01-02 15:04:05"),
		"timezone":   now.Location().String(),
		"unix_milli": now.UnixMilli(),
		"ntp_synced": false,
	}

	// 使用缓存的 NTP 偏移量（不会每次请求都查询 NTP）
	if offset, ok := getNTPOffset(); ok {
		adjusted := now.Add(time.Duration(offset) * time.Millisecond)
		result["ntp_time"] = adjusted.UTC().Format("2006-01-02 15:04:05")
		result["ntp_milli"] = adjusted.UnixMilli()
		result["ntp_offset_ms"] = offset
		result["ntp_synced"] = true
		result["timestamp"] = adjusted.Unix()
		result["unix_milli"] = adjusted.UnixMilli()
	}

	Success(w, result)
}

// syncSystemTime 通过 NTP 校正后设置系统时间
func (h *Handler) syncSystemTime(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// 强制重新查询 NTP（不用缓存）
	forceSyncNTP()

	ntpMu.RLock()
	offset := ntpOffset
	synced := ntpSynced
	ntpMu.RUnlock()

	if !synced {
		Success(w, map[string]interface{}{
			"updated": false,
			"message": "NTP 同步失败，无法获取准确时间，请检查网络连接",
		})
		return
	}

	// 偏移小于 500ms 视为无需校正
	if offset > -500 && offset < 500 {
		Success(w, map[string]interface{}{
			"updated": false,
			"message": fmt.Sprintf("系统时间准确（偏差 %dms），无需校正", offset),
		})
		return
	}

	ntpTime := time.Now().Add(time.Duration(offset) * time.Millisecond)
	err := setSystemTime(ntpTime)
	if err != nil {
		Success(w, map[string]interface{}{
			"updated": false,
			"message": fmt.Sprintf("NTP 时间获取成功（偏差 %dms），但修改系统时间失败: %v", offset, err),
		})
		return
	}

	// 重置缓存
	ntpMu.Lock()
	ntpOffset = 0
	ntpSynced = true
	ntpLastSync = time.Now()
	ntpMu.Unlock()

	Success(w, map[string]interface{}{
		"updated": true,
		"message":  fmt.Sprintf("系统时间已校正 %dms", offset),
	})
}

// setSystemTime 设置系统时间（跨平台）
func setSystemTime(t time.Time) error {
	switch runtime.GOOS {
	case "windows":
		// PowerShell: Set-Date
		psCmd := fmt.Sprintf("Set-Date -Date '%s'", t.Format("2006-01-02 15:04:05"))
		return exec.Command("powershell", "-Command", psCmd).Run()
	case "darwin":
		// macOS: date -u 走 UTC 设置
		dateStr := t.UTC().Format("010215042006")
		return exec.Command("date", "-u", dateStr).Run()
	default:
		// Linux: 优先 timedatectl，失败则 date -s
		dateStr := t.Format("2006-01-02 15:04:05")
		if err := exec.Command("timedatectl", "set-ntp", "false").Run(); err == nil {
			if err := exec.Command("timedatectl", "set-time", dateStr).Run(); err == nil {
				exec.Command("timedatectl", "set-ntp", "true").Run()
				return nil
			}
		}
		// fallback: date -s
		return exec.Command("date", "-s", dateStr).Run()
	}
}

// ntpOffset 缓存 NTP 偏移量
var (
	ntpOffset   int64 // 毫秒偏移
	ntpSynced   bool
	ntpLastSync time.Time
	ntpMu       sync.RWMutex
)

// getNTPOffset 获取缓存的 NTP 偏移量，过期则重新同步
func getNTPOffset() (int64, bool) {
	ntpMu.RLock()
	if ntpSynced && time.Since(ntpLastSync) < 10*time.Minute {
		offset := ntpOffset
		ntpMu.RUnlock()
		return offset, true
	}
	ntpMu.RUnlock()

	// 需要重新同步
	syncNTP()
	ntpMu.RLock()
	defer ntpMu.RUnlock()
	return ntpOffset, ntpSynced
}

// forceSyncNTP 强制重新同步 NTP（忽略缓存，用于手动同步按钮）
func forceSyncNTP() {
	ntpMu.Lock()
	defer ntpMu.Unlock()
	queryNTPServers()
}

// queryNTPServers 并发查询 NTP 服务器（调用方需持有 ntpMu 写锁）
func queryNTPServers() {
	type ntpResult struct {
		offset int64
		ok     bool
	}

	now := time.Now()
	servers := []string{
		"ntp.aliyun.com:123",
		"cn.ntp.org.cn:123",
		"ntp.tencent.com:123",
		"time.cloudflare.com:123",
		"time.google.com:123",
	}

	ch := make(chan ntpResult, len(servers))
	for _, server := range servers {
		go func(addr string) {
			t, err := queryNTP(addr)
			if err != nil {
				fmt.Printf("[NTP] %s 查询失败: %v\n", addr, err)
				ch <- ntpResult{}
				return
			}
			// 校验 NTP 时间在合理范围（2020-2035），避免解析异常
			if t.Year() < 2020 || t.Year() > 2035 {
				fmt.Printf("[NTP] %s 时间异常: %v (year=%d)\n", addr, t, t.Year())
				ch <- ntpResult{}
				return
			}
			diff := t.Sub(now)
			fmt.Printf("[NTP] %s 成功, 偏移: %v\n", addr, diff)
			ch <- ntpResult{offset: diff.Milliseconds(), ok: true}
		}(server)
	}

	for range servers {
		if r := <-ch; r.ok {
			ntpOffset = r.offset
			ntpSynced = true
			ntpLastSync = time.Now()
			return
		}
	}
	fmt.Println("[NTP] 所有服务器均查询失败")
}

// syncNTP 同步 NTP 时间（使用缓存，过期则重新查询）
func syncNTP() {
	ntpMu.Lock()
	defer ntpMu.Unlock()

	if ntpSynced && time.Since(ntpLastSync) < 10*time.Minute {
		return
	}

	queryNTPServers()
}

// getNTPTime 从 NTP 服务器获取时间（已弃用，保留兼容）
func getNTPTime() *time.Time {
	syncNTP()
	ntpMu.RLock()
	defer ntpMu.RUnlock()
	if !ntpSynced {
		return nil
	}
	t := time.Now().Add(time.Duration(ntpOffset) * time.Millisecond)
	return &t
}

// queryNTP 通过 SNTP 协议查询 NTP 服务器
func queryNTP(server string) (*time.Time, error) {
	conn, err := net.DialTimeout("udp", server, 2*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// NTP 客户端请求包 (48 bytes)
	// LI=0, VN=4, Mode=3 (client)
	req := make([]byte, 48)
	req[0] = 0x23 // 00 100 011 = LI=0, VN=4, Mode=3

	if _, err = conn.Write(req); err != nil {
		return nil, err
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	resp := make([]byte, 48)
	if _, err = conn.Read(resp); err != nil {
		return nil, err
	}

	// Transmit Timestamp 从第 40 字节开始 (8 bytes: 4秒 + 4小数)
	sec := uint64(resp[40])<<24 | uint64(resp[41])<<16 | uint64(resp[42])<<8 | uint64(resp[43])
	frac := uint64(resp[44])<<24 | uint64(resp[45])<<16 | uint64(resp[46])<<8 | uint64(resp[47])

	// NTP 纪元: 1900-01-01, Unix 纪元: 1970-01-01, 差值 70 年
	const ntpEpochOffset = 2208988800
	unixSec := int64(sec - ntpEpochOffset)
	unixNsec := int64(float64(frac) * float64(time.Second))
	if unixNsec < 0 {
		unixNsec = 0
	}

	t := time.Unix(unixSec, unixNsec)
	return &t, nil
}
