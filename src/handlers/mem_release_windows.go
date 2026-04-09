//go:build windows

package handlers

import (
	"syscall"
)

var procEmptyWorkingSet = modpsapi.NewProc("EmptyWorkingSet")

// FreeSystemMemory 释放系统级内存（Windows）
// 使用 EmptyWorkingSet API 清空进程工作集
func FreeSystemMemory() error {
	handle, _ := syscall.GetCurrentProcess()
	ret, _, err := procEmptyWorkingSet.Call(uintptr(handle))
	if ret == 0 {
		return err
	}
	return nil
}
