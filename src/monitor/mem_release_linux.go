//go:build linux

package monitor

import (
	"os"
	"syscall"
)

// FreeSystemMemory 释放系统级内存（Linux）
// 调用 sync 刷写脏页 + 写入 drop_caches 清除页面缓存
func FreeSystemMemory() error {
	syscall.Sync()
	return os.WriteFile("/proc/sys/vm/drop_caches", []byte("3"), 0200)
}
