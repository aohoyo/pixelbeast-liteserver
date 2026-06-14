//go:build windows

package monitor

import "syscall"

// modpsapi psapi.dll 句柄，供 mem_windows.go 与 mem_release_windows.go 共用。
// 提取到独立文件避免在同一包内重复声明。
var modpsapi = syscall.NewLazyDLL("psapi.dll")
