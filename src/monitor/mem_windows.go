//go:build windows

package monitor

import (
	"runtime"
	"syscall"
	"unsafe"
)

var (
	modpsapi                 = syscall.NewLazyDLL("psapi.dll")
	procGetProcessMemoryInfo = modpsapi.NewProc("GetProcessMemoryInfo")
)

// PROCESS_MEMORY_COUNTERS Windows 内存计数器结构
type PROCESS_MEMORY_COUNTERS struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
}

// GetProcessMemory 获取进程内存使用（Windows 版本）
func GetProcessMemory() uint64 {
	if mem := getProcessMemoryOS(); mem > 0 {
		return mem
	}
	return getProcessMemoryFallback()
}

// getProcessMemoryOS 使用 Windows API 获取进程内存
func getProcessMemoryOS() uint64 {
	handle, _ := syscall.GetCurrentProcess()
	var memCounters PROCESS_MEMORY_COUNTERS
	memCounters.CB = uint32(unsafe.Sizeof(memCounters))

	ret, _, _ := procGetProcessMemoryInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&memCounters)),
		uintptr(memCounters.CB),
	)
	if ret == 0 {
		return 0
	}

	return uint64(memCounters.WorkingSetSize)
}

// getProcessMemoryFallback 使用 runtime.MemStats 估算
func getProcessMemoryFallback() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapInuse + m.StackInuse + m.MSpanInuse + m.MCacheInuse
}
