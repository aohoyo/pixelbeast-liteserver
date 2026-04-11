package admin

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	psnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"

	"pixelbeast/src/config"
	"pixelbeast/src/handlers"
)

var startTime = time.Now()

// cpuHistory 存储 CPU 历史数据用于趋势图
var cpuHistory []float64

// 后台 CPU 采样
var (
	lastCPUPercent float64
	lastCPUPerCore []float64
	cpuMu          sync.RWMutex
)

// 后台网络采样
var (
	netSpeedSentKB float64
	netSpeedRecvKB float64
	netTotalSentGB float64
	netTotalRecvGB float64
	lastNetSent    uint64
	lastNetRecv    uint64
	lastNetTime    time.Time
	netMu          sync.RWMutex
)

// 后台磁盘IO采样
var (
	diskIOSpeedWriteKB float64
	diskIOSpeedReadKB  float64
	diskIOIOPS         float64
	diskIOLatencyMs    float64
	diskIOTotalWriteGB float64
	diskIOTotalReadGB  float64
	lastDiskWrite      uint64
	lastDiskRead       uint64
	lastDiskWriteCount uint64
	lastDiskReadCount  uint64
	lastDiskReadTime   uint64
	lastDiskWriteTime  uint64
	lastDiskIOTime     time.Time
	diskIOMu           sync.RWMutex
)

func init() {
	go func() {
		for {
			// 总体使用率
			percent, err := cpu.Percent(time.Second, false)
			cpuMu.Lock()
			if err == nil && len(percent) > 0 {
				lastCPUPercent = percent[0]
			}
			cpuMu.Unlock()

			// 每核使用率
			perCore, err := cpu.Percent(0, true)
			if err == nil && len(perCore) > 0 {
				cpuMu.Lock()
				lastCPUPerCore = perCore
				cpuMu.Unlock()
			}

			time.Sleep(2 * time.Second)
		}
	}()

	// 网络 & 磁盘IO 采样
	go func() {
		for {
			// 网络速率
			counters, err := psnet.IOCounters(false)
			if err == nil && len(counters) > 0 {
				c := counters[0]
				now := time.Now()
				netMu.Lock()
				if !lastNetTime.IsZero() {
					elapsed := now.Sub(lastNetTime).Seconds()
					if elapsed > 0 {
						netSpeedSentKB = float64(c.BytesSent-lastNetSent) / 1024 / elapsed
						netSpeedRecvKB = float64(c.BytesRecv-lastNetRecv) / 1024 / elapsed
					}
				}
				lastNetSent = c.BytesSent
				lastNetRecv = c.BytesRecv
				lastNetTime = now
				netTotalSentGB = float64(c.BytesSent) / 1024 / 1024 / 1024
				netTotalRecvGB = float64(c.BytesRecv) / 1024 / 1024 / 1024
				netMu.Unlock()
			}

			// 磁盘IO速率
			ioCounters, err := disk.IOCounters()
			if err == nil && len(ioCounters) > 0 {
				// 固定选取第一个设备（按名称排序，确保每次一致）
				var keys []string
				for k := range ioCounters {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				c := ioCounters[keys[0]]

				now := time.Now()
				diskIOMu.Lock()
				if !lastDiskIOTime.IsZero() {
					elapsed := now.Sub(lastDiskIOTime).Seconds()
					if elapsed > 0 {
						diskIOSpeedWriteKB = float64(c.WriteBytes-lastDiskWrite) / 1024 / elapsed
						diskIOSpeedReadKB = float64(c.ReadBytes-lastDiskRead) / 1024 / elapsed
						diskIOIOPS = float64((c.WriteCount-lastDiskWriteCount)+(c.ReadCount-lastDiskReadCount)) / elapsed
						totalOps := (c.WriteCount - lastDiskWriteCount) + (c.ReadCount - lastDiskReadCount)
						if totalOps > 0 {
							totalTime := (c.WriteTime - lastDiskWriteTime) + (c.ReadTime - lastDiskReadTime)
							diskIOLatencyMs = float64(totalTime) / float64(totalOps)
						} else {
							diskIOLatencyMs = 0
						}
					}
				}
				lastDiskWrite = c.WriteBytes
				lastDiskRead = c.ReadBytes
				lastDiskWriteCount = c.WriteCount
				lastDiskReadCount = c.ReadCount
				lastDiskWriteTime = c.WriteTime
				lastDiskReadTime = c.ReadTime
				lastDiskIOTime = now
				diskIOTotalWriteGB = float64(c.WriteBytes) / 1024 / 1024 / 1024
				diskIOTotalReadGB = float64(c.ReadBytes) / 1024 / 1024 / 1024
				diskIOMu.Unlock()
			}

			time.Sleep(2 * time.Second)
		}
	}()
}

// ==================== 状态 ====================

func (h *Handler) getStatus(w http.ResponseWriter, r *http.Request) {
	processMem := handlers.GetProcessMemory()

	// 获取服务器状态
	adminRunning := false
	sitesRunning := false
	adminPort := h.ConfigManager.Server.Admin.Port

	if h.ServerManager != nil {
		adminRunning = h.ServerManager.IsAdminRunning()
		sitesRunning = h.ServerManager.IsSitesRunning()
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
	serverUptime := time.Since(startTime)
	statusData := map[string]interface{}{
		"memory_mb":       processMem / 1024 / 1024,
		"goroutines":      runtime.NumGoroutine(),
		"go_version":      runtime.Version(),
		"os":              runtime.GOOS,
		"arch":            runtime.GOARCH,
		"os_name":         getOSName(),
		"os_name_short":   getOSShortName(),
		"kernel":          getKernelVersion(),
		"hostname":        getHostname(),
		"server_uptime":   formatUptime(serverUptime),
		"server_uptime_ms": serverUptime.Milliseconds(),
		"system_uptime":     formatUptime(getSystemUptime()),
			"system_uptime_ms":  getSystemUptime().Milliseconds(),
		"admin_running":   adminRunning,
		"admin_port":      adminPort,
		"sites_running":   sitesRunning,
		"sites_enabled":   countEnabledSites(h.ConfigManager.Sites.Sites),
		"sites_count":     len(h.ConfigManager.Sites.Sites),
		"sites":           sites,
	}

	// 如果有 CSRF token，添加到响应中
	if csrfToken != "" {
		statusData["csrf_token"] = csrfToken
	}

	Success(w, statusData)
}

// ==================== 系统监控 API ====================

func (h *Handler) getSystemStatus(w http.ResponseWriter, r *http.Request) {
	// 获取进程内存
	processMem := handlers.GetProcessMemory()
	memoryMB := float64(processMem) / 1024 / 1024

	// 获取真实的 CPU 信息（从后台采样读取，不阻塞）
	cpuMu.RLock()
	cpuPercent := lastCPUPercent
	cpuMu.RUnlock()
	cpuCores, _ := cpu.Counts(false)
	if cpuCores == 0 {
		cpuCores = runtime.NumCPU()
	}
	cpuThreads, _ := cpu.Counts(true)
	if cpuThreads == 0 {
		cpuThreads = cpuCores
	}
	cpuModel := getCPUModel()

	// 更新 CPU 历史记录
	cpuHistory = append(cpuHistory, cpuPercent)
	if len(cpuHistory) > 5 {
		cpuHistory = cpuHistory[len(cpuHistory)-5:]
	}

	// 获取 FTP 服务状态

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

	// Swap 内存
	swapInfo, _ := mem.SwapMemory()
	var swapPercent, swapUsedGB, swapTotalGB float64
	if swapInfo != nil && swapInfo.Total > 0 {
		swapPercent = swapInfo.UsedPercent
		swapUsedGB = float64(swapInfo.Used) / 1024 / 1024 / 1024
		swapTotalGB = float64(swapInfo.Total) / 1024 / 1024 / 1024
	}

	// 获取所有磁盘分区信息
	programDir, _ := os.Getwd()
	partitions, _ := disk.Partitions(false)
	type diskEntry struct {
		Mount   string  `json:"mount"`
		Device  string  `json:"device"`
		Fstype  string  `json:"fstype"`
		TotalGB float64 `json:"total_gb"`
		UsedGB  float64 `json:"used_gb"`
		FreeGB  float64 `json:"free_gb"`
		Percent float64 `json:"percent"`
	}
	var disks []diskEntry
	var primaryDisk *diskEntry

	for _, p := range partitions {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil || usage == nil || usage.Total == 0 {
			continue
		}
		disks = append(disks, diskEntry{
			Mount:   p.Mountpoint,
			Device:  p.Device,
			Fstype:  p.Fstype,
			TotalGB: float64(usage.Total) / 1024 / 1024 / 1024,
			UsedGB:  float64(usage.Used) / 1024 / 1024 / 1024,
			FreeGB:  float64(usage.Free) / 1024 / 1024 / 1024,
			Percent: usage.UsedPercent,
		})
	}
	// 找到程序所在磁盘（最长匹配挂载点）
	bestLen := 0
	for i := range disks {
		if strings.HasPrefix(programDir, disks[i].Mount) && len(disks[i].Mount) > bestLen {
			primaryDisk = &disks[i]
			bestLen = len(disks[i].Mount)
		}
	}
	if primaryDisk == nil && len(disks) > 0 {
		primaryDisk = &disks[0]
	}

	// 获取真实的负载信息
	loadAvg, _ := load.Avg()
	load1m, load5m, load15m := 0.0, 0.0, 0.0
	if loadAvg != nil {
		load1m = loadAvg.Load1
		load5m = loadAvg.Load5
		load15m = loadAvg.Load15
	}

	// 获取进程数量（活跃/总数）
	var processActive int
	allPids, _ := process.Pids()
	for _, pid := range allPids {
		p, err := process.NewProcess(pid)
		if err != nil {
			continue
		}
		statuses, _ := p.Status()
		for _, s := range statuses {
			if s == "R" {
				processActive++
				break
			}
		}
	}
	processTotal := len(allPids)

	// 读取每核 CPU 使用率
	cpuMu.RLock()
	cpuPerCore := make([]float64, len(lastCPUPerCore))
	copy(cpuPerCore, lastCPUPerCore)
	cpuMu.RUnlock()
	// 构建状态数据
	statusData := map[string]interface{}{
		// CPU
		"cpu_percent":  cpuPercent,
		"cpu_cores":    cpuCores,
		"cpu_threads":  cpuThreads,
		"cpu_model":    cpuModel,
		"cpu_history":  cpuHistory,
		"cpu_per_core": cpuPerCore,

		// 内存
		"memory_percent":       memPercent,
		"memory_used_gb":       memUsedGB,
		"memory_total_gb":      memTotalGB,
		"memory_free_gb":       memFreeGB,
		"memory_available_gb":  memAvailableGB,
		"memory_shared_mb":     memSharedMB,
		"memory_buff_cache_mb": memBuffCacheMB,

		// Swap
		"swap_percent":  swapPercent,
		"swap_used_gb":  swapUsedGB,
		"swap_total_gb": swapTotalGB,

		// 磁盘（所有磁盘合计）
		"disk_percent": func() float64 {
			var total, used float64
			for _, d := range disks {
				total += d.TotalGB
				used += d.UsedGB
			}
			if total > 0 {
				return used / total * 100
			}
			return 0
		}(),
		"disk_used_gb": func() float64 {
			var sum float64
			for _, d := range disks {
				sum += d.UsedGB
			}
			return sum
		}(),
		"disk_total_gb": func() float64 {
			var sum float64
			for _, d := range disks {
				sum += d.TotalGB
			}
			return sum
		}(),
		"disk_free_gb": func() float64 {
			var sum float64
			for _, d := range disks {
				sum += d.FreeGB
			}
			return sum
		}(),
		"disk_mount": func() string {
			if primaryDisk != nil {
				return primaryDisk.Mount
			}
			return "/"
		}(),
		"disk_filesystem": func() string {
			if primaryDisk != nil {
				return primaryDisk.Fstype
			}
			return ""
		}(),
		"disk_type": getDiskType(),
		"disks":     disks,

		// 负载
		"load_avg":       []float64{load1m, load5m, load15m},
		"process_active": processActive,
		"process_total":  processTotal,

		// 运行时间
		"uptime_str":        formatUptime(time.Since(startTime)),
		"server_uptime_ms":  time.Since(startTime).Milliseconds(),
		"system_uptime":     formatUptime(getSystemUptime()),
		"system_uptime_ms":  getSystemUptime().Milliseconds(),

		// 网络
		"net_sent_rate_kb":  netSpeedSentKB,
		"net_recv_rate_kb":  netSpeedRecvKB,
		"net_total_sent_gb": netTotalSentGB,
		"net_total_recv_gb": netTotalRecvGB,

		// 磁盘IO
		"diskio_speed_write_kb": diskIOSpeedWriteKB,
		"diskio_speed_read_kb":  diskIOSpeedReadKB,
		"diskio_total_write_gb": diskIOTotalWriteGB,
		"diskio_total_read_gb":  diskIOTotalReadGB,
		"diskio_iops":           diskIOIOPS,
		"diskio_latency_ms":     diskIOLatencyMs,

		// 服务状态
		"admin_running": h.ServerManager != nil && h.ServerManager.IsAdminRunning(),
		"admin_port":    h.ConfigManager.Server.Admin.Port,
		"sites_running": h.ServerManager != nil && h.ServerManager.IsSitesRunning(),
		"sites_enabled": countEnabledSites(h.ConfigManager.Sites.Sites),
		"sites_count":   len(h.ConfigManager.Sites.Sites),
		"ftp_running":   h.ServerManager != nil && h.ServerManager.IsFTPRunning(),
		"ftp_port":      h.ConfigManager.FTP.Port,
		"ftp_users":     len(h.ConfigManager.FTP.Users),

		// 保留原有字段
		"memory_mb":     memoryMB,
		"goroutines":    runtime.NumGoroutine(),
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"os_name":       getOSName(),
		"os_name_short": getOSShortName(),
		"kernel":        getKernelVersion(),
		"hostname":      getHostname(),
	}

	Success(w, statusData)
}

// formatUptime 格式化运行时长
func formatUptime(d time.Duration) string {
	totalSeconds := int(d.Seconds())
	days := totalSeconds / 86400
	hours := (totalSeconds % 86400) / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	if days > 0 {
		return fmt.Sprintf("%d天 %d时 %d分 %d秒", days, hours, minutes, seconds)
	}
	if hours > 0 {
		return fmt.Sprintf("%d时 %d分 %d秒", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%d分 %d秒", minutes, seconds)
	}
	return fmt.Sprintf("%d秒", seconds)
}

// getSystemUptime 获取系统运行时长
// Windows 实现见 uptime_windows.go (GetTickCount64)
// Linux/macOS 实现见 uptime_other.go (host.Info BootTime)

// countEnabledSites 统计启用的站点数量
func countEnabledSites(sites []config.SiteConfig) int {
	count := 0
	for _, s := range sites {
		if s.Enabled {
			count++
		}
	}
	return count
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

// getOSName 获取完整操作系统名称（如 Debian GNU/Linux 12 (bookworm)）
var osNameCache string
var osNameOnce sync.Once

// getOSShortName 获取简短操作系统名称（如 Debian 12）
var osNameShortCache string
var osNameShortOnce sync.Once

func getOSName() string {
	osNameOnce.Do(func() {
		switch runtime.GOOS {
		case "linux":
			osNameCache = readLinuxDistro()
		case "darwin":
			osNameCache = readMacOSName()
		case "windows":
			osNameCache = readWindowsName()
		default:
			osNameCache = runtime.GOOS
		}
	})
	return osNameCache
}

func readLinuxDistro() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "Linux"
	}
	lines := strings.Split(string(data), "\n")
	var name, version string
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		val := strings.Trim(parts[1], "\"'")
		switch parts[0] {
		case "PRETTY_NAME":
			if val != "" {
				return val
			}
		case "NAME":
			name = val
		case "VERSION":
			version = val
		case "VERSION_ID":
			if version == "" {
				version = val
			}
		}
	}
	if name != "" {
		if version != "" {
			return name + " " + version
		}
		return name
	}
	return "Linux"
}

func getOSShortName() string {
	osNameShortOnce.Do(func() {
		switch runtime.GOOS {
		case "linux":
			osNameShortCache = readLinuxDistroShort()
		case "darwin":
			osNameShortCache = readMacOSNameShort()
		case "windows":
			info, err := host.Info()
			if err == nil && info.Platform != "" {
				osNameShortCache = info.Platform
				if info.PlatformVersion != "" {
					parts := strings.SplitN(info.PlatformVersion, ".", 2)
					osNameShortCache += " " + parts[0]
				}
			} else {
				osNameShortCache = "Windows"
			}
		default:
			osNameShortCache = runtime.GOOS
		}
	})
	return osNameShortCache
}

func readLinuxDistroShort() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "Linux"
	}
	lines := strings.Split(string(data), "\n")
	var id, versionID string
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		val := strings.Trim(parts[1], "\"'")
		switch parts[0] {
		case "ID":
			id = strings.ToLower(val)
		case "VERSION_ID":
			versionID = val
		}
	}

	nameMap := map[string]string{
		"debian":              "Debian",
		"ubuntu":              "Ubuntu",
		"centos":              "CentOS",
		"fedora":              "Fedora",
		"arch":                "Arch",
		"opensuse-leap":       "openSUSE",
		"opensuse-tumbleweed": "openSUSE",
		"alpine":              "Alpine",
		"raspbian":            "Raspbian",
		"linuxmint":           "Linux Mint",
		"manjaro":             "Manjaro",
		"amzn":                "Amazon Linux",
		"rocky":               "Rocky Linux",
		"almalinux":           "AlmaLinux",
		"gentoo":              "Gentoo",
		"void":                "Void Linux",
		"nixos":               "NixOS",
	}

	name := id
	if mapped, ok := nameMap[id]; ok {
		name = mapped
	} else if len(id) > 0 {
		name = strings.ToUpper(id[:1]) + id[1:]
	}

	if versionID != "" {
		return name + " " + versionID
	}
	return name
}

func readMacOSNameShort() string {
	out, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		return "macOS"
	}
	ver := strings.TrimSpace(string(out))
	// 只取主版本号，如 "14.2.1" → "macOS 14"
	parts := strings.SplitN(ver, ".", 2)
	if len(parts) > 0 {
		return "macOS " + parts[0]
	}
	return "macOS"
}

func readMacOSName() string {
	out, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		return "macOS"
	}
	return "macOS " + strings.TrimSpace(string(out))
}

func readWindowsName() string {
	info, err := host.Info()
	if err != nil {
		return "Windows"
	}
	name := info.Platform
	if info.PlatformVersion != "" {
		name += " " + info.PlatformVersion
	}
	if name == "" {
		return "Windows"
	}
	return name
}

// getKernelVersion 获取内核版本
func getKernelVersion() string {
	switch runtime.GOOS {
	case "windows":
		info, err := host.Info()
		if err != nil {
			return ""
		}
		return info.KernelVersion
	default:
		out, err := exec.Command("uname", "-r").Output()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
}

// getHostname 获取主机名
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return ""
	}
	return hostname
}

// getDiskType 获取磁盘类型
func getDiskType() string {
	// gopsutil 不能直接获取磁盘类型（SSD/HDD）
	// 可以通过检测是否为旋转磁盘来判断
	// 这里返回简化版本
	return "--"
}
func (h *Handler) freeMemory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	// 1. 记录释放前的内存
	beforeProcess := handlers.GetProcessMemory()
	memInfoBefore, _ := mem.VirtualMemory()
	var beforeSystemUsed float64
	if memInfoBefore != nil {
		beforeSystemUsed = float64(memInfoBefore.Used)
	}

	// 2. Go 运行时释放：GC + 归还 OS
	runtime.GC()
	debug.FreeOSMemory()

	// 3. 平台特定的系统内存释放
	osErr := handlers.FreeSystemMemory()
	osCacheCleared := osErr == nil

	// 4. 记录释放后的内存
	afterProcess := handlers.GetProcessMemory()
	memInfoAfter, _ := mem.VirtualMemory()
	var afterSystemUsed float64
	if memInfoAfter != nil {
		afterSystemUsed = float64(memInfoAfter.Used)
	}

	// 5. 计算释放量
	processFreedMB := float64(int64(beforeProcess)-int64(afterProcess)) / 1024 / 1024
	if processFreedMB < 0 {
		processFreedMB = 0
	}
	systemFreedMB := (beforeSystemUsed - afterSystemUsed) / 1024 / 1024
	if systemFreedMB < 0 {
		systemFreedMB = 0
	}
	totalFreedMB := processFreedMB + systemFreedMB

	username := h.getSessionUsername(r)
	handlers.LogPanelOperation(handlers.LogLevelInfo, "[系统] 释放内存: %.1f MB (用户=%s)", totalFreedMB, username)

	Success(w, map[string]interface{}{
		"freed_mb":         totalFreedMB,
		"process_freed_mb": processFreedMB,
		"system_freed_mb":  systemFreedMB,
		"before_mb":        float64(beforeProcess) / 1024 / 1024,
		"after_mb":         float64(afterProcess) / 1024 / 1024,
		"os_cache_cleared": osCacheCleared,
		"message":          fmt.Sprintf("已释放 %.1f MB 内存（进程 %.1f MB + 系统 %.1f MB）", totalFreedMB, processFreedMB, systemFreedMB),
	})
}

func (h *Handler) scanCleanup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	items := h.scanSystemJunk()

	totalMB := 0.0
	for _, item := range items {
		if !item.NeedAdmin {
			totalMB += item.SizeMB
		}
	}

	Success(w, map[string]interface{}{
		"items":    items,
		"total_mb": totalMB,
	})
}

func (h *Handler) executeCleanup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	// 解析选中的清理项
	var req struct {
		Items []string `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON: "+err.Error())
		return
	}

	if len(req.Items) == 0 {
		Success(w, map[string]interface{}{
			"cleaned_mb": 0,
			"message":    "未选择清理项",
		})
		return
	}

	selected := make(map[string]bool, len(req.Items))
	for _, item := range req.Items {
		selected[item] = true
	}

	cleanedMB := h.cleanSystemJunk(selected)

	username := h.getSessionUsername(r)
	handlers.LogPanelConfigChange(username, fmt.Sprintf("系统清理 %.1f MB", cleanedMB), true)

	Success(w, map[string]interface{}{
		"cleaned_mb": cleanedMB,
		"message":    fmt.Sprintf("已清理 %.1f MB", cleanedMB),
	})
}

// junkItem 垃圾文件扫描项
type junkItem struct {
	Name      string  `json:"name"`
	Desc      string  `json:"desc"`
	SizeMB    float64 `json:"size_mb"`
	Count     int     `json:"count"`
	Path      string  `json:"path"`
	Cleanable bool    `json:"cleanable"`
	NeedAdmin bool    `json:"need_admin"` // 需要管理员权限
}

// scanSystemJunk 扫描系统垃圾文件
func (h *Handler) scanSystemJunk() []junkItem {
	var items []junkItem

	switch runtime.GOOS {
	case "linux":
		items = h.scanLinuxJunk()
	case "darwin":
		items = h.scanDarwinJunk()
	case "windows":
		items = h.scanWindowsJunk()
	}

	return items
}

// scanLinuxJunk 扫描 Linux 系统垃圾
func (h *Handler) scanLinuxJunk() []junkItem {
	var items []junkItem

	// 系统临时文件 /tmp
	if sizeMB, count := scanDirOlderThan("/tmp", 24*time.Hour); count > 0 {
		items = append(items, junkItem{
			Name: "system_tmp", Desc: "系统临时文件 (/tmp)",
			SizeMB: sizeMB, Count: count, Path: "/tmp", Cleanable: true,
		})
	}

	// /var/tmp
	if sizeMB, count := scanDirOlderThan("/var/tmp", 24*time.Hour); count > 0 {
		items = append(items, junkItem{
			Name: "var_tmp", Desc: "持久临时文件 (/var/tmp)",
			SizeMB: sizeMB, Count: count, Path: "/var/tmp", Cleanable: true,
		})
	}

	// 包管理器缓存
	for _, cacheDir := range []string{"/var/cache/apt/archives", "/var/cache/yum", "/var/cache/dnf"} {
		if sizeMB, count := calcDirSize(cacheDir); count > 0 {
			name := "pkg_cache"
			desc := "包管理器缓存"
			if strings.Contains(cacheDir, "apt") {
				desc = "APT 缓存 (/var/cache/apt)"
			} else if strings.Contains(cacheDir, "yum") {
				desc = "YUM 缓存 (/var/cache/yum)"
			} else {
				desc = "DNF 缓存 (/var/cache/dnf)"
			}
			items = append(items, junkItem{
				Name: name, Desc: desc,
				SizeMB: sizeMB, Count: count, Path: cacheDir, Cleanable: true,
			})
			break // 只取第一个存在的
		}
	}

	// 缩略图缓存
	if home, err := os.UserHomeDir(); err == nil {
		thumbDir := filepath.Join(home, ".cache", "thumbnails")
		if sizeMB, count := calcDirSize(thumbDir); count > 0 {
			items = append(items, junkItem{
				Name: "thumbnails", Desc: "缩略图缓存",
				SizeMB: sizeMB, Count: count, Path: thumbDir, Cleanable: true,
			})
		}
	}

	// Crash reports / core dumps
	for _, crashDir := range []string{"/var/crash", "/var/lib/systemd/coredump"} {
		if sizeMB, count := calcDirSize(crashDir); count > 0 {
			items = append(items, junkItem{
				Name: "crash_dumps", Desc: "崩溃转储文件",
				SizeMB: sizeMB, Count: count, Path: crashDir, Cleanable: true,
			})
			break
		}
	}

	// 回收站
	if home, err := os.UserHomeDir(); err == nil {
		trashDir := filepath.Join(home, ".local", "share", "Trash", "files")
		if sizeMB, count := calcDirSize(trashDir); count > 0 {
			items = append(items, junkItem{
				Name: "trash", Desc: "回收站",
				SizeMB: sizeMB, Count: count, Path: trashDir, Cleanable: true,
			})
		}
	}

	// npm 缓存
	if home, err := os.UserHomeDir(); err == nil {
		npmCache := filepath.Join(home, ".npm", "_cacache")
		if sizeMB, count := calcDirSize(npmCache); count > 0 {
			items = append(items, junkItem{
				Name: "npm_cache", Desc: "npm 缓存",
				SizeMB: sizeMB, Count: count, Path: npmCache, Cleanable: true,
			})
		}
	}

	// pip 缓存
	if home, err := os.UserHomeDir(); err == nil {
		pipCache := filepath.Join(home, ".cache", "pip")
		if sizeMB, count := calcDirSize(pipCache); count > 0 {
			items = append(items, junkItem{
				Name: "pip_cache", Desc: "pip 缓存",
				SizeMB: sizeMB, Count: count, Path: pipCache, Cleanable: true,
			})
		}
	}

	return items
}

// scanDarwinJunk 扫描 macOS 系统垃圾
func (h *Handler) scanDarwinJunk() []junkItem {
	var items []junkItem

	// 系统临时目录
	for _, tmpDir := range []string{"/tmp", "/private/tmp", "/var/folders"} {
		if sizeMB, count := scanDirOlderThan(tmpDir, 24*time.Hour); count > 0 {
			items = append(items, junkItem{
				Name: "system_tmp", Desc: "系统临时文件",
				SizeMB: sizeMB, Count: count, Path: tmpDir, Cleanable: true,
			})
			break
		}
	}

	// 用户缓存
	if home, err := os.UserHomeDir(); err == nil {
		cacheDir := filepath.Join(home, "Library", "Caches")
		if sizeMB, count := calcDirSize(cacheDir); count > 0 {
			items = append(items, junkItem{
				Name: "user_cache", Desc: "用户缓存 (Library/Caches)",
				SizeMB: sizeMB, Count: count, Path: cacheDir, Cleanable: true,
			})
		}
	}

	// 废纸篓
	if home, err := os.UserHomeDir(); err == nil {
		trashDir := filepath.Join(home, ".Trash")
		if sizeMB, count := calcDirSize(trashDir); count > 0 {
			items = append(items, junkItem{
				Name: "trash", Desc: "废纸篓",
				SizeMB: sizeMB, Count: count, Path: trashDir, Cleanable: true,
			})
		}
	}

	// Xcode 编译缓存
	if home, err := os.UserHomeDir(); err == nil {
		xcodeDir := filepath.Join(home, "Library", "Developer", "Xcode", "DerivedData")
		if sizeMB, count := calcDirSize(xcodeDir); count > 0 {
			items = append(items, junkItem{
				Name: "xcode_derived", Desc: "Xcode 编译缓存",
				SizeMB: sizeMB, Count: count, Path: xcodeDir, Cleanable: true,
			})
		}
	}

	// npm 缓存
	if home, err := os.UserHomeDir(); err == nil {
		npmCache := filepath.Join(home, ".npm", "_cacache")
		if sizeMB, count := calcDirSize(npmCache); count > 0 {
			items = append(items, junkItem{
				Name: "npm_cache", Desc: "npm 缓存",
				SizeMB: sizeMB, Count: count, Path: npmCache, Cleanable: true,
			})
		}
	}

	// pip 缓存
	if home, err := os.UserHomeDir(); err == nil {
		pipCache := filepath.Join(home, ".cache", "pip")
		if sizeMB, count := calcDirSize(pipCache); count > 0 {
			items = append(items, junkItem{
				Name: "pip_cache", Desc: "pip 缓存",
				SizeMB: sizeMB, Count: count, Path: pipCache, Cleanable: true,
			})
		}
	}

	return items
}

// scanWindowsJunk 扫描 Windows 系统垃圾
func (h *Handler) scanWindowsJunk() []junkItem {
	var items []junkItem

	// Windows Temp
	admin := isAdmin()
	for _, tmpDir := range []string{os.Getenv("TEMP"), os.Getenv("TMP"), `C:\Windows\Temp`} {
		if tmpDir == "" {
			continue
		}
		// C:\Windows\Temp 需要管理员权限
		needAdmin := !admin && strings.HasPrefix(tmpDir, `C:\Windows`)
		if sizeMB, count := scanDirOlderThan(tmpDir, 24*time.Hour); count > 0 {
			items = append(items, junkItem{
				Name: "system_tmp", Desc: "系统临时文件",
				SizeMB: sizeMB, Count: count, Path: tmpDir, Cleanable: !needAdmin, NeedAdmin: needAdmin,
			})
			break
		}
	}

	// Windows Update 缓存（需要管理员权限）
	if sizeMB, count := calcDirSize(`C:\Windows\SoftwareDistribution\Download`); count > 0 {
		items = append(items, junkItem{
			Name: "update_cache", Desc: "Windows 更新缓存",
			SizeMB: sizeMB, Count: count, Path: `C:\Windows\SoftwareDistribution\Download`,
			Cleanable: admin, NeedAdmin: !admin,
		})
	}

	// 崩溃转储
	crashDir := os.Getenv("LOCALAPPDATA") + `\CrashDumps`
	if crashDir != `\CrashDumps` {
		if sizeMB, count := calcDirSize(crashDir); count > 0 {
			items = append(items, junkItem{
				Name: "crash_dumps", Desc: "崩溃转储文件",
				SizeMB: sizeMB, Count: count, Path: crashDir, Cleanable: true,
			})
		}
	}

	// 缩略图缓存
	thumbDir := os.Getenv("LOCALAPPDATA") + `\Microsoft\Windows\Explorer`
	if thumbDir != `\Microsoft\Windows\Explorer` {
		if sizeMB, count := scanFilesByExt(thumbDir, ".db"); count > 0 {
			items = append(items, junkItem{
				Name: "thumb_cache", Desc: "缩略图缓存",
				SizeMB: sizeMB, Count: count, Path: thumbDir, Cleanable: true,
			})
		}
	}

	// 预读取缓存（需要管理员权限）
	if sizeMB, count := calcDirSize(`C:\Windows\Prefetch`); count > 0 {
		items = append(items, junkItem{
			Name: "prefetch", Desc: "预读取缓存",
			SizeMB: sizeMB, Count: count, Path: `C:\Windows\Prefetch`,
			Cleanable: admin, NeedAdmin: !admin,
		})
	}

	// 回收站
	if out, err := exec.Command("powershell", "-Command",
		"(New-Object -ComObject Shell.Application).NameSpace(0xA).Items().Count").Output(); err == nil {
		if strings.TrimSpace(string(out)) != "0" {
			items = append(items, junkItem{
				Name: "recycle_bin", Desc: "回收站",
				SizeMB: 0, Count: -1, Path: "Recycle Bin", Cleanable: true,
			})
		}
	}

	return items
}

// cleanSystemJunk 执行系统垃圾清理
func (h *Handler) cleanSystemJunk(selected map[string]bool) float64 {
	var totalCleaned float64

	switch runtime.GOOS {
	case "linux":
		totalCleaned += h.cleanLinuxJunk(selected)
	case "darwin":
		totalCleaned += h.cleanDarwinJunk(selected)
	case "windows":
		totalCleaned += h.cleanWindowsJunk(selected)
	}

	return totalCleaned
}

func (h *Handler) cleanLinuxJunk(selected map[string]bool) float64 {
	var cleaned float64

	// 清理临时文件（尝试 sudo，失败则直接清理）
	if selected["system_tmp"] {
		cleaned += sudoCleanDirOlderThan("/tmp", 24*time.Hour)
	}
	if selected["var_tmp"] {
		cleaned += sudoCleanDirOlderThan("/var/tmp", 24*time.Hour)
	}

	// 清理包管理器缓存
	if selected["pkg_cache"] {
		// 先计算缓存大小，清理后差值即为清理量
		var beforeMB float64
		cacheDir := ""
		for _, d := range []string{"/var/cache/apt/archives", "/var/cache/yum", "/var/cache/dnf"} {
			if mb, _ := calcDirSize(d); mb > 0 {
				beforeMB = mb
				cacheDir = d
				break
			}
		}
		if cacheDir != "" {
			if _, err := exec.LookPath("apt-get"); err == nil {
				exec.Command("sudo", "apt-get", "clean", "-y").Run()
			} else if _, err := exec.LookPath("yum"); err == nil {
				exec.Command("sudo", "yum", "clean", "all", "-y").Run()
			} else if _, err := exec.LookPath("dnf"); err == nil {
				exec.Command("sudo", "dnf", "clean", "all", "-y").Run()
			}
			afterMB, _ := calcDirSize(cacheDir)
			if afterMB < beforeMB {
				cleaned += beforeMB - afterMB
			}
		}
	}

	if selected["thumbnails"] {
		if home, err := os.UserHomeDir(); err == nil {
			thumbDir := filepath.Join(home, ".cache", "thumbnails")
			cleaned += sudoCleanDirContents(thumbDir)
		}
	}

	if selected["crash_dumps"] {
		for _, dir := range []string{"/var/crash", "/var/lib/systemd/coredump"} {
			cleaned += sudoCleanDirContents(dir)
		}
	}

	if selected["trash"] {
		if home, err := os.UserHomeDir(); err == nil {
			trashDir := filepath.Join(home, ".local", "share", "Trash", "files")
			cleaned += sudoCleanDirContents(trashDir)
		}
	}

	if selected["npm_cache"] {
		if home, err := os.UserHomeDir(); err == nil {
			npmCache := filepath.Join(home, ".npm", "_cacache")
			cleaned += sudoCleanDirContents(npmCache)
		}
	}

	if selected["pip_cache"] {
		if home, err := os.UserHomeDir(); err == nil {
			pipCache := filepath.Join(home, ".cache", "pip")
			cleaned += sudoCleanDirContents(pipCache)
		}
	}

	return cleaned
}

func (h *Handler) cleanDarwinJunk(selected map[string]bool) float64 {
	var cleaned float64

	if selected["system_tmp"] {
		cleaned += sudoCleanDirOlderThan("/tmp", 24*time.Hour)
		cleaned += sudoCleanDirOlderThan("/private/tmp", 24*time.Hour)
	}

	if selected["user_cache"] {
		if home, err := os.UserHomeDir(); err == nil {
			cacheDir := filepath.Join(home, "Library", "Caches")
			cleaned += sudoCleanDirContents(cacheDir)
		}
	}

	if selected["trash"] {
		if home, err := os.UserHomeDir(); err == nil {
			trashDir := filepath.Join(home, ".Trash")
			cleaned += sudoCleanDirContents(trashDir)
		}
	}

	if selected["xcode_derived"] {
		if home, err := os.UserHomeDir(); err == nil {
			xcodeDir := filepath.Join(home, "Library", "Developer", "Xcode", "DerivedData")
			cleaned += sudoCleanDirContents(xcodeDir)
		}
	}

	if selected["npm_cache"] {
		if home, err := os.UserHomeDir(); err == nil {
			npmCache := filepath.Join(home, ".npm", "_cacache")
			cleaned += sudoCleanDirContents(npmCache)
		}
	}

	if selected["pip_cache"] {
		if home, err := os.UserHomeDir(); err == nil {
			pipCache := filepath.Join(home, ".cache", "pip")
			cleaned += sudoCleanDirContents(pipCache)
		}
	}

	return cleaned
}

func (h *Handler) cleanWindowsJunk(selected map[string]bool) float64 {
	var cleaned float64

	if selected["system_tmp"] {
		for _, tmpDir := range []string{os.Getenv("TEMP"), os.Getenv("TMP"), `C:\Windows\Temp`} {
			if tmpDir != "" {
				cleaned += sudoCleanDirOlderThan(tmpDir, 24*time.Hour)
			}
		}
	}

	if selected["update_cache"] {
		// 停止 Windows Update 服务，释放文件锁
		runCmdTimeout("net", 10*time.Second, "stop", "wuauserv")
		cleaned += sudoCleanDirContents(`C:\Windows\SoftwareDistribution\Download`)
		runCmdTimeout("net", 10*time.Second, "start", "wuauserv")
	}

	if selected["crash_dumps"] {
		crashDir := os.Getenv("LOCALAPPDATA") + "\\CrashDumps"
		if crashDir != "\\CrashDumps" {
			cleaned += sudoCleanDirContents(crashDir)
		}
	}

	if selected["thumb_cache"] {
		thumbDir := os.Getenv("LOCALAPPDATA") + "\\Microsoft\\Windows\\Explorer"
		if thumbDir != "\\Microsoft\\Windows\\Explorer" {
			cleaned += cleanFilesByExt(thumbDir, ".db")
		}
	}

	if selected["prefetch"] {
		// 停止 SysMain 服务，释放文件锁
		runCmdTimeout("net", 10*time.Second, "stop", "SysMain")
		cleaned += sudoCleanDirContents(`C:\Windows\Prefetch`)
		runCmdTimeout("net", 10*time.Second, "start", "SysMain")
	}

	if selected["recycle_bin"] {
		runCmdTimeout("powershell", 30*time.Second, "-Command", "Clear-RecycleBin", "-Force")
	}

	return cleaned
}

// ==================== 垃圾扫描/清理工具函数 ====================

// cleanFailedItem 清理失败的文件/目录项
type cleanFailedItem struct {
	path  string
	size  float64
	isDir bool
}

// sudoCleanDirOlderThan 尝试清理目录中超过指定时间的文件（优先提权）
func sudoCleanDirOlderThan(dir string, age time.Duration) float64 {
	cutoff := time.Now().Add(-age)
	var cleaned float64
	var failed []cleanFailedItem

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !info.ModTime().Before(cutoff) {
			return nil
		}
		sizeMB := float64(info.Size()) / 1024 / 1024

		// 先尝试直接删除
		if os.Remove(path) == nil {
			cleaned += sizeMB
			return nil
		}
		// 记录失败项，稍后批量提权删除
		failed = append(failed, cleanFailedItem{path, sizeMB, false})
		return nil
	})

	cleaned += sudoBatchClean(failed)
	return cleaned
}

// sudoCleanDirContents 清空目录内容（优先提权）
func sudoCleanDirContents(dir string) float64 {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	var cleaned float64
	var failed []cleanFailedItem

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		// 计算实际大小（包括子目录）
		var sizeMB float64
		if entry.IsDir() {
			sizeMB, _ = calcDirSize(path)
		} else {
			if info, err := entry.Info(); err == nil {
				sizeMB = float64(info.Size()) / 1024 / 1024
			}
		}
		// 先尝试直接删除
		var delErr error
		if entry.IsDir() {
			delErr = os.RemoveAll(path)
		} else {
			delErr = os.Remove(path)
		}
		if delErr == nil {
			cleaned += sizeMB
			continue
		}
		// 记录失败项，稍后批量提权删除
		failed = append(failed, cleanFailedItem{path, sizeMB, entry.IsDir()})
	}

	cleaned += sudoBatchClean(failed)
	return cleaned
}

// sudoBatchClean 批量提权删除文件/目录（Windows: 单条 PowerShell; Linux: sudo）
func sudoBatchClean(items []cleanFailedItem) float64 {
	if len(items) == 0 {
		return 0
	}
	if runtime.GOOS == "windows" {
		// 构建一条 PowerShell 命令批量删除
		var paths []string
		for _, item := range items {
			paths = append(paths, fmt.Sprintf("'%s'", item.path))
		}
		cmd := fmt.Sprintf("Remove-Item -Path %s -Force -ErrorAction SilentlyContinue", strings.Join(paths, ","))
		runCmdTimeout("powershell", 30*time.Second, "-Command", cmd)
		// 检查哪些已删除，统计清理量
		var cleaned float64
		for _, item := range items {
			if _, err := os.Stat(item.path); err != nil {
				cleaned += item.size
			}
		}
		return cleaned
	}
	// Linux/macOS: 逐个 sudo 删除
	var cleaned float64
	for _, item := range items {
		if item.isDir {
			if exec.Command("sudo", "rm", "-rf", item.path).Run() == nil {
				cleaned += item.size
			}
		} else {
			if exec.Command("sudo", "rm", "-f", item.path).Run() == nil {
				cleaned += item.size
			}
		}
	}
	return cleaned
}

// runCmdTimeout 带超时执行命令，超时则终止进程
func runCmdTimeout(name string, timeout time.Duration, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Start()
	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		cmd.Process.Kill()
	}
}

// calcDirSize 计算目录总大小和文件数
func calcDirSize(dir string) (totalMB float64, count int) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		totalMB += float64(info.Size()) / 1024 / 1024
		count++
		return nil
	})
	return
}

// scanDirOlderThan 扫描目录中超过指定时间的文件大小
func scanDirOlderThan(dir string, age time.Duration) (totalMB float64, count int) {
	cutoff := time.Now().Add(-age)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			totalMB += float64(info.Size()) / 1024 / 1024
			count++
		}
		return nil
	})
	return
}

// cleanDirOlderThan 清理目录中超过指定时间的文件
func cleanDirOlderThan(dir string, age time.Duration) float64 {
	cutoff := time.Now().Add(-age)
	var cleaned float64
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			sizeMB := float64(info.Size()) / 1024 / 1024
			if os.Remove(path) == nil {
				cleaned += sizeMB
			}
		}
		return nil
	})
	return cleaned
}

// cleanDirContents 清空目录内容（保留目录本身）
func cleanDirContents(dir string) float64 {
	var cleaned float64
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}
		sizeMB := float64(info.Size()) / 1024 / 1024
		var err2 error
		if entry.IsDir() {
			err2 = os.RemoveAll(path)
		} else {
			err2 = os.Remove(path)
		}
		if err2 == nil {
			cleaned += sizeMB
		}
	}
	return cleaned
}

// cleanFilesByExt 清理目录中指定扩展名的文件
// scanFilesByExt 扫描目录中指定扩展名的文件大小
func scanFilesByExt(dir, ext string) (totalMB float64, count int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ext) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		totalMB += float64(info.Size()) / 1024 / 1024
		count++
	}
	return
}

func cleanFilesByExt(dir, ext string) float64 {
	var cleaned float64
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ext) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		if os.Remove(path) == nil {
			cleaned += float64(info.Size()) / 1024 / 1024
		}
	}
	return cleaned
}

// truncateAppLogs 截断应用当前日志
func (h *Handler) truncateAppLogs() float64 {
	logDir := handlers.GetLogBaseDir()
	if logDir == "" {
		return 0
	}
	var cleaned float64
	categories := []string{"http", "ftp", "panel"}
	for _, cat := range categories {
		dir := filepath.Join(logDir, cat)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasSuffix(name, ".log") {
				continue
			}
			baseName := strings.TrimSuffix(name, ".log")
			if strings.Contains(baseName, ".") {
				continue
			}
			filePath := filepath.Join(dir, name)
			info, err := entry.Info()
			if err != nil {
				continue
			}
			sizeMB := float64(info.Size()) / 1024 / 1024
			if os.WriteFile(filePath, []byte{}, 0644) == nil {
				cleaned += sizeMB
			}
		}
	}
	return cleaned
}

// parseSizeToMB 解析大小字符串为 MB（如 "3.2G", "500M", "1024K"）
func parseSizeToMB(s string) float64 {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return 0
	}
	unit := strings.ToUpper(s[len(s)-1:])
	numStr := s[:len(s)-1]
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}
	switch unit {
	case "G", "GB":
		return num * 1024
	case "M", "MB":
		return num
	case "K", "KB":
		return num / 1024
	case "B":
		return num / 1024 / 1024
	}
	return 0
}

// ==================== 配置 ====================

func (h *Handler) getConfig(w http.ResponseWriter, r *http.Request) {
	// 序列化 ServerConfig，直接返回与 server.json 一致的结构
	cfg := *h.ConfigManager.Server
	cfg.Admin.Password = "" // 不返回密码
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

	// 记住密码，反序列化后恢复
	oldPassword := h.ConfigManager.Server.Admin.Password

	// JSON 反序列化：直接将前端数据映射到 ServerConfig
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		BadRequest(w, "JSON 解析失败: "+err.Error())
		handlers.LogPanelConfigChange(username, "保存配置", false)
		return
	}

	var newCfg config.ServerConfig
	if err := json.Unmarshal(jsonBytes, &newCfg); err != nil {
		BadRequest(w, "配置格式错误: "+err.Error())
		handlers.LogPanelConfigChange(username, "保存配置", false)
		return
	}

	// 密码特殊处理：前端传空则保留原密码
	if newCfg.Admin.Password != "" {
		h.ConfigManager.SetAdminPassword(newCfg.Admin.Password)
	} else {
		newCfg.Admin.Password = oldPassword
	}

	*h.ConfigManager.Server = newCfg

	// 解析 sites 配置（单独的配置文件）
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
				siteConfigs = append(siteConfigs, siteConfig)
			}
		}
		h.ConfigManager.Sites.Sites = siteConfigs
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

	if h.ConfigManager == nil {
		InternalServerError(w, "配置管理器未初始化")
		return
	}

	// 使用 ConfigManager 统一的默认配置方法重置
	h.ConfigManager.ResetToDefaults()

	if err := h.ConfigManager.Save(); err != nil {
		InternalServerError(w, err.Error())
		handlers.LogPanelConfigChange(username, "重置配置", false)
		return
	}

	handlers.LogPanelConfigChange(username, "重置配置", true)

	// 返回重置后的完整配置
	cfg := *h.ConfigManager.Server
	cfg.Admin.Password = ""
	Success(w, cfg)
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

// syncSystemTime 通过 NTP 校正后设置系统时间，同时返回当前时间数据
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

	// 构建时间数据（无论同步是否成功都返回）
	now := time.Now()
	tz := h.ConfigManager.Server.Timezone
	if tz == "" {
		tz = "Asia/Shanghai"
	}
	loc, _ := time.LoadLocation(tz)
	result := map[string]interface{}{
		"timestamp":  now.Unix(),
		"time":       now.In(loc).Format("2006-01-02 15:04:05"),
		"utc_time":   now.UTC().Format("2006-01-02 15:04:05"),
		"timezone":   tz,
		"unix_milli": now.UnixMilli(),
		"ntp_synced": synced,
		"updated":    false,
	}

	if !synced {
		result["message"] = "NTP 同步失败，无法获取准确时间，请检查网络连接"
		Success(w, result)
		return
	}

	// 偏移小于 500ms 视为无需校正
	if offset > -500 && offset < 500 {
		result["message"] = fmt.Sprintf("系统时间准确（偏差 %dms），无需校正", offset)
		Success(w, result)
		return
	}

	ntpTime := time.Now().Add(time.Duration(offset) * time.Millisecond)
	err := setSystemTime(ntpTime)
	if err != nil {
		result["message"] = fmt.Sprintf("NTP 时间获取成功（偏差 %dms），但修改系统时间失败: %v", offset, err)
		Success(w, result)
		return
	}

	// 重置缓存
	ntpMu.Lock()
	ntpOffset = 0
	ntpSynced = true
	ntpLastSync = time.Now()
	ntpMu.Unlock()

	// 同步成功后重新获取当前时间
	now = time.Now()
	result["timestamp"] = now.Unix()
	result["time"] = now.Format("2006-01-02 15:04:05")
	result["utc_time"] = now.UTC().Format("2006-01-02 15:04:05")
	result["unix_milli"] = now.UnixMilli()
	result["ntp_synced"] = true
	result["updated"] = true
	result["message"] = fmt.Sprintf("系统时间已校正 %dms", offset)

	Success(w, result)
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

// getNTPTime 从 NTP 服务器获取时间（已弃用，保留兼容）

// queryNTP 通过 SNTP 协议查询 NTP 服务器
func queryNTP(server string) (*time.Time, error) {
	conn, err := net.DialTimeout("udp", server, 2*time.Second)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// NTP 客户端请求: LI=0, VN=4, Mode=3
	req := make([]byte, 48)
	req[0] = 0x23

	if _, err = conn.Write(req); err != nil {
		return nil, err
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	resp := make([]byte, 48)
	// 用 io.ReadFull 确保读取完整 48 字节
	if _, err = io.ReadFull(conn, resp); err != nil {
		return nil, err
	}

	// 校验响应：VN=4, Mode=4 (服务器)
	vn := (resp[0] >> 3) & 0x07
	mode := resp[0] & 0x07
	if vn != 4 || mode != 4 {
		return nil, fmt.Errorf("无效 NTP 响应 (VN=%d, Mode=%d)", vn, mode)
	}

	// 校验 Stratum（1=主服务器, 2-15=次级服务器）
	stratum := resp[1]
	if stratum == 0 || stratum > 15 {
		return nil, fmt.Errorf("无效 Stratum=%d", stratum)
	}

	// Transmit Timestamp: bytes 40-47 (4秒 + 4小数)
	sec := uint64(resp[40])<<24 | uint64(resp[41])<<16 | uint64(resp[42])<<8 | uint64(resp[43])
	frac := uint64(resp[44])<<24 | uint64(resp[45])<<16 | uint64(resp[46])<<8 | uint64(resp[47])

	// NTP 纪元 1900 → Unix 纪元 1970
	const ntpEpochOffset uint64 = 2208988800
	if sec < ntpEpochOffset {
		return nil, fmt.Errorf("NTP 时间戳异常 (sec=%d)", sec)
	}
	unixSec := int64(sec - ntpEpochOffset)
	unixNsec := int64(float64(frac) / float64(1<<32) * float64(time.Second))

	t := time.Unix(unixSec, unixNsec)
	return &t, nil
}

// ==================== 备份管理 ====================

// listBackups 列出备份文件
func (h *Handler) listBackups(w http.ResponseWriter, r *http.Request) {
	backupDir := h.ConfigManager.GetBackupDir()
	absPath := resolvePath(backupDir)

	entries, err := os.ReadDir(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			Success(w, map[string]interface{}{
				"backups": []interface{}{},
				"dir":     backupDir,
			})
			return
		}
		InternalServerError(w, "读取备份目录失败: "+err.Error())
		return
	}

	backups := make([]map[string]interface{}, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		name := entry.Name()
		if !(strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".zip")) {
			continue
		}
		backups = append(backups, map[string]interface{}{
			"name":     name,
			"size":     info.Size(),
			"modified": info.ModTime().Format("2006-01-02 15:04:05"),
		})
	}

	// 按修改时间倒序
	for i, j := 0, len(backups)-1; i < j; i, j = i+1, j-1 {
		backups[i], backups[j] = backups[j], backups[i]
	}

	Success(w, map[string]interface{}{
		"backups": backups,
		"dir":     backupDir,
	})
}

// createBackup 手动创建备份
func (h *Handler) createBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	backupDir := h.ConfigManager.GetBackupDir()
	absBackupDir := resolvePath(backupDir)
	os.MkdirAll(absBackupDir, 0755)

	timestamp := time.Now().Format("2006-01-02_150405")
	backupName := fmt.Sprintf("backup_%s.tar.gz", timestamp)
	backupPath := filepath.Join(absBackupDir, backupName)

	// 根据 items 配置决定备份内容
	items := h.ConfigManager.Server.Backup.Items
	if len(items) == 0 {
		items = []string{"config"}
	}

	// 创建 tar.gz，包含多个目录
	f, err := os.Create(backupPath)
	if err != nil {
		InternalServerError(w, "创建备份失败: "+err.Error())
		return
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, item := range items {
		var srcDir, prefix string
		switch item {
		case "config":
			srcDir = resolvePath(h.ConfigManager.ConfigDir())
			prefix = "config"
		case "sites":
			srcDir = resolvePath(h.ConfigManager.GetSitesDir())
			prefix = "sites"
		case "ftp":
			srcDir = resolvePath(h.ConfigManager.GetFTPRoot())
			prefix = "ftp"
		default:
			continue
		}

		if _, err := os.Stat(srcDir); os.IsNotExist(err) {
			continue
		}

		filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			relPath, err := filepath.Rel(srcDir, path)
			if err != nil || relPath == "." {
				return nil
			}
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return nil
			}
			header.Name = filepath.Join(prefix, relPath)
			if info.IsDir() {
				header.Name += "/"
			}
			if err := tw.WriteHeader(header); err != nil {
				return nil
			}
			if !info.IsDir() {
				file, err := os.Open(path)
				if err != nil {
					return nil
				}
				defer file.Close()
				io.Copy(tw, file)
			}
			return nil
		})
	}

	username := h.getSessionUsername(r)
	handlers.LogPanelConfigChange(username, "创建备份 "+backupName, true)
	Success(w, map[string]interface{}{
		"name":    backupName,
		"message": "备份创建成功",
	})

}

// deleteBackup 删除备份
func (h *Handler) deleteBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}

	name, ok := data["name"].(string)
	if !ok || name == "" {
		BadRequest(w, "备份文件名不能为空")
		return
	}

	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		BadRequest(w, "无效的备份文件名")
		return
	}

	backupDir := h.ConfigManager.GetBackupDir()
	absPath := filepath.Join(resolvePath(backupDir), name)

	if err := os.Remove(absPath); err != nil {
		InternalServerError(w, "删除备份失败: "+err.Error())
		return
	}
	username := h.getSessionUsername(r)

	handlers.LogPanelConfigChange(username, "删除备份 "+name, true)
	SuccessMessage(w, "备份已删除")
}

// downloadBackup 下载备份文件
func (h *Handler) downloadBackup(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		BadRequest(w, "缺少备份文件名")
		return
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		BadRequest(w, "无效的备份文件名")
		return
	}
	backupDir := h.ConfigManager.GetBackupDir()
	absPath := filepath.Join(resolvePath(backupDir), name)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		BadRequest(w, "备份文件不存在")
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, absPath)
}

// restoreBackup 从备份恢复
func (h *Handler) restoreBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid JSON")
		return
	}
	if req.Name == "" {
		BadRequest(w, "备份文件名不能为空")
		return
	}
	if strings.Contains(req.Name, "/") || strings.Contains(req.Name, "\\") || strings.Contains(req.Name, "..") {
		BadRequest(w, "无效的备份文件名")
		return
	}

	backupDir := h.ConfigManager.GetBackupDir()
	absPath := filepath.Join(resolvePath(backupDir), req.Name)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		BadRequest(w, "备份文件不存在")
		return
	}

	configDir := h.ConfigManager.ConfigDir()

	tmpDir, err := os.MkdirTemp("", "pixelbeast-restore-*")
	if err != nil {
		InternalServerError(w, "创建临时目录失败")
		return
	}
	defer os.RemoveAll(tmpDir)

	f, err := os.Open(absPath)
	if err != nil {
		InternalServerError(w, "打开备份文件失败")
		return
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		InternalServerError(w, "解压失败")
		return
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			InternalServerError(w, "读取备份失败")
			return
		}
		if strings.HasPrefix(header.Name, "/") || strings.Contains(header.Name, "..") {
			continue
		}
		target := filepath.Join(tmpDir, header.Name)
		if header.Typeflag == tar.TypeDir {
			os.MkdirAll(target, os.FileMode(header.Mode))
			continue
		}
		if header.Typeflag == tar.TypeReg {
			os.MkdirAll(filepath.Dir(target), 0755)
			out, err := os.Create(target)
			if err != nil {
				continue
			}
			io.Copy(out, tr)
			out.Close()
		}
	}

	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(tmpDir, path)
		dst := filepath.Join(configDir, rel)
		os.MkdirAll(filepath.Dir(dst), 0755)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		os.WriteFile(dst, data, info.Mode())
		return nil
	})

	username := h.getSessionUsername(r)
	handlers.LogPanelConfigChange(username, "从备份恢复 "+req.Name, true)
	SuccessMessage(w, "备份恢复成功，重新加载配置生效")
}

// createTarGz 创建 tar.gz 压缩包
func createTarGz(outputPath, srcDir, prefix string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		header.Name = filepath.Join(prefix, relPath)
		if info.IsDir() {
			header.Name += "/"
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(tw, f)
			return err
		}

		return nil
	})
}

// ==================== 系统操作 ====================

// restartServer 重启服务进程
func (h *Handler) restartServer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		MethodNotAllowed(w, "Method not allowed")
		return
	}

	username := h.getSessionUsername(r)
	handlers.LogPanelRuntime(handlers.LogLevelInfo, "[系统] 服务器重启 (用户=%s)", username)

	SuccessMessage(w, "服务正在重启...")

	go func() {
		time.Sleep(500 * time.Millisecond)
		exec.Command(os.Args[0], os.Args[1:]...).Start()
		os.Exit(0)
	}()
}

// checkUpdate 检查 GitHub 最新版本
func (h *Handler) checkUpdate(w http.ResponseWriter, r *http.Request) {
	currentVersion := h.Version
	if currentVersion == "" {
		currentVersion = "v0.0.0"
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/aohoyo/pixelbeast-liteserver/releases/latest")
	if err != nil {
		Success(w, map[string]interface{}{
			"current_version": currentVersion,
			"latest_version":  "",
			"has_update":      false,
			"message":         "无法连接更新服务器",
		})
		return
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		Success(w, map[string]interface{}{
			"current_version": currentVersion,
			"latest_version":  "",
			"has_update":      false,
			"message":         "解析更新信息失败",
		})
		return
	}

	latestVersion := release.TagName
	hasUpdate := latestVersion != currentVersion && latestVersion != ""

	Success(w, map[string]interface{}{
		"current_version": currentVersion,
		"latest_version":  latestVersion,
		"has_update":      hasUpdate,
		"changelog":       release.Body,
		"download_url":    release.HTMLURL,
		"message": func() string {
			if hasUpdate {
				return "发现新版本: " + latestVersion
			}
			return "已是最新版本"
		}(),
	})
}
