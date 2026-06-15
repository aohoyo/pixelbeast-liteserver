//go:build windows

package monitor

import (
	"syscall"
	"unsafe"
)

var (
	modkernel32               = syscall.NewLazyDLL("kernel32.dll")
	procEmptyWorkingSet       = modpsapi.NewProc("EmptyWorkingSet")
	procSetSystemFileCacheSize = modkernel32.NewProc("SetSystemFileCacheSize")
)

// FreeSystemMemory 释放系统级内存（Windows）
// 1. 清空当前进程工作集
// 2. 尝试清理系统文件缓存
func FreeSystemMemory() error {
	// 1. 清空当前进程工作集
	handle, _ := syscall.GetCurrentProcess()
	ret, _, err := procEmptyWorkingSet.Call(uintptr(handle))
	if ret == 0 {
		return err
	}

	// 2. 尝试限制系统文件缓存（需要 SE_INCREASE_QUOTA_NAME 权限，失败忽略）
	// 设置文件缓存最小=0, 最大=1，强制释放缓存
	procSetSystemFileCacheSize.Call(0, 1, 0)

	return nil
}

// FreeSystemMemoryEx 增强版内存释放
func FreeSystemMemoryEx() error {
	// 使用更激进的内存清理策略
	handle, _ := syscall.GetCurrentProcess()

	// 清空工作集
	procEmptyWorkingSet.Call(uintptr(handle))

	// 限制文件缓存
	procSetSystemFileCacheSize.Call(0, 1, 0)

	return nil
}

// needed for unused import safety
var _ = unsafe.Pointer(nil)
