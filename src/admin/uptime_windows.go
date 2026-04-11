package admin

import (
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procGetTickCount64  = kernel32.NewProc("GetTickCount64")
	winSessionStartOnce sync.Once
	winSessionStart     time.Time
)

// getSystemUptime 获取 Windows 系统真实运行时长
// 通过 Event Log 获取最近一次启动/恢复的时间（解决快速启动导致 BootTime 不准确的问题）
func getSystemUptime() time.Duration {
	winSessionStartOnce.Do(func() {
		winSessionStart = readWindowsSessionStart()
	})
	if winSessionStart.IsZero() {
		// 回退：使用 GetTickCount64
		ret, _, _ := procGetTickCount64.Call()
		return time.Duration(ret) * time.Millisecond
	}
	return time.Since(winSessionStart)
}

// readWindowsSessionStart 从 Event Log 获取最近一次启动/恢复时间
// 优先级：Event ID 107（从睡眠恢复）> Event ID 12（完整启动）> GetTickCount64 回退
func readWindowsSessionStart() time.Time {
	// 查询最近一次完整启动 (Event ID 12) 和从睡眠恢复 (Event ID 107)，取最近的时间
	ps := `$e1 = Get-WinEvent -FilterHashtable @{LogName='System'; ProviderName='Microsoft-Windows-Kernel-General'; ID=12} -MaxEvents 1 -ErrorAction SilentlyContinue; $e2 = Get-WinEvent -FilterHashtable @{LogName='System'; ProviderName='Microsoft-Windows-Kernel-Power'; ID=107} -MaxEvents 1 -ErrorAction SilentlyContinue; $t = $null; if ($e1) { $t = $e1.TimeCreated }; if ($e2 -and (!$t -or $e2.TimeCreated -gt $t)) { $t = $e2.TimeCreated }; if ($t) { $t.ToString('yyyy-MM-ddTHH:mm:ss') }`

	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps).Output()
	if err != nil {
		return fallbackBootTime()
	}

	s := strings.TrimSpace(string(out))
	if s == "" {
		return fallbackBootTime()
	}

	t, err := time.ParseInLocation("2006-01-02T15:04:05", s, time.Local)
	if err != nil {
		return fallbackBootTime()
	}

	// 校验：如果获取的时间在未来或超过 365 天前，视为异常
	since := time.Since(t)
	if since < 0 || since > 365*24*time.Hour {
		return fallbackBootTime()
	}

	return t
}

// fallbackBootTime 回退方案：用 GetTickCount64 推算启动时间
func fallbackBootTime() time.Time {
	ret, _, _ := procGetTickCount64.Call()
	uptime := time.Duration(ret) * time.Millisecond
	if uptime <= 0 {
		return time.Time{}
	}
	return time.Now().Add(-uptime)
}
