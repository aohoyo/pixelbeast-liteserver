//go:build !windows

package monitor

import (
	"os"
	"runtime"
	"strconv"
	"strings"
)

// GetProcessMemory 获取进程内存使用（跨平台）
func GetProcessMemory() uint64 {
	if mem := getProcessMemoryOS(); mem > 0 {
		return mem
	}
	return getProcessMemoryFallback()
}

// getProcessMemoryOS 平台特定实现
func getProcessMemoryOS() uint64 {
	if runtime.GOOS == "linux" {
		return getProcessMemoryLinux()
	}
	return 0 // 其他平台返回 0，使用 fallback
}

// getProcessMemoryLinux 从 /proc/self/status 读取 VmRSS
func getProcessMemoryLinux() uint64 {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, _ := strconv.ParseUint(fields[1], 10, 64)
				return kb * 1024
			}
		}
	}
	return 0
}

// getProcessMemoryFallback 使用 runtime.MemStats 估算
func getProcessMemoryFallback() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapInuse + m.StackInuse + m.MSpanInuse + m.MCacheInuse
}
