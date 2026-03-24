//go:build windows

package handlers

import (
	"golang.org/x/sys/windows"
	"unsafe"
)

// getProcessMemoryOS 使用 Windows API 获取进程内存
func getProcessMemoryOS() uint64 {
	handle := windows.CurrentProcess()
	var memCounters windows.PROCESS_MEMORY_COUNTERS
	memCounters.Cb = uint32(unsafe.Sizeof(memCounters))

	err := windows.GetProcessMemoryInfo(handle, &memCounters, memCounters.Cb)
	if err != nil {
		return 0
	}

	return memCounters.WorkingSetSize
}